package service

import (
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/sastromikus/pip_shortener/internal/repository"
)

const (
	idLen      = 8
	stringbase = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// URLRepository is the subset of repository operations required by Shortener.
type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

// Shortener implements URL shortening business logic.
type Shortener struct {
	repo     repository.URLRepository
	deleteCh chan DeleteTask
}

// DeleteTask represents a request to delete multiple short URLs for a user.
type DeleteTask struct {
	UserID string
	IDs    []string
}

// NewShortener creates a new Shortener using the provided repository.
func NewShortener(repo repository.URLRepository) *Shortener {
	s := &Shortener{repo: repo, deleteCh: make(chan DeleteTask, 1024)}

	return s
}

// Shorten creates or returns a short id for the provided URL.
func (s *Shortener) Shorten(raw string) (string, error) {
	id, _, err := s.ShortenWithExisting(raw)
	return id, err
}

func (s *Shortener) ShortenWithExisting(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, errors.New("empty url")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	if pg, ok := s.repo.(*repository.PostgresRepository); ok {
		const (
			originalUQ = "urls_original_url_uq"
			shortUQ    = "urls_short_id_key"
		)

		for tries := 0; tries < 10; tries++ {
			id, err := s.generateUniqueID(idLen, 10)
			if err != nil {
				return "", false, err
			}

			err = pg.Insert(id, raw)
			if err == nil {
				return id, false, nil
			}

			if repository.IsUniqueViolationOn(err, originalUQ) {
				existing, ok := pg.GetByOriginal(raw)
				if ok {
					return existing, true, nil
				}
				return "", false, err
			}

			if repository.IsUniqueViolationOn(err, shortUQ) {
				continue
			}

			return "", false, err
		}
		return "", false, errors.New("cannot insert url")
	}

	id, err := s.generateUniqueID(idLen, 10)
	if err != nil {
		return "", false, err
	}
	s.repo.Put(id, raw)

	return id, false, nil
}

// ShortenForUser creates a short URL and associates it with user if userID is provided.
func (s *Shortener) ShortenForUser(raw, userID string) (string, bool, error) {
	id, existed, err := s.ShortenWithExisting(raw)
	if err != nil {
		return "", false, err
	}

	if userID != "" {
		if us, ok := s.repo.(interface {
			AddUserURL(userID, shortID string) error
		}); ok {
			_ = us.AddUserURL(userID, id)
		}
	}

	return id, existed, nil
}

// ListUserURLs returns all URLs created by the given user.
func (s *Shortener) ListUserURLs(userID string) ([]repository.UserURL, error) {
	us, ok := s.repo.(interface {
		ListUserURLs(userID string) ([]repository.UserURL, error)
	})
	if !ok {
		return nil, nil
	}

	return us.ListUserURLs(userID)
}

func (s *Shortener) Resolve(id string) (string, bool) {
	return s.repo.Get(id)
}

func (s *Shortener) generateUniqueID(length int, tries int) (string, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", err
		}
		if !s.repo.Exists(id) {
			return id, nil
		}
	}
	return "", errors.New("could not generate unique id")
}

func randomstringbase(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("invalid length")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = stringbase[int(buf[i])%len(stringbase)]
	}
	return string(out), nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("unsupported scheme")
	}
	if u.Host == "" {
		return errors.New("empty host")
	}
	return nil
}

// EnqueueDelete queues an asynchronous delete request.
func (s *Shortener) EnqueueDelete(userID string, ids []string) {
	if userID == "" || len(ids) == 0 {
		return
	}

	select {
	case s.deleteCh <- DeleteTask{UserID: userID, IDs: ids}:
	default:
	}
}

// StartDeleteWorker starts background processing of delete tasks.
func (s *Shortener) StartDeleteWorker(batchSize int, flushEvery time.Duration) {
	pg, ok := s.repo.(*repository.PostgresRepository)
	if !ok {
		return
	}

	if batchSize <= 0 {
		batchSize = 64
	}
	if flushEvery <= 0 {
		flushEvery = 500 * time.Millisecond
	}

	go func() {
		type bucket struct {
			ids map[string]struct{}
		}

		pending := make(map[string]map[string]struct{})

		flush := func() {
			for userID, set := range pending {
				if len(set) == 0 {
					continue
				}
				ids := make([]string, 0, len(set))
				for id := range set {
					ids = append(ids, id)
				}
				_ = pg.MarkDeleted(userID, ids)
				delete(pending, userID)
			}
		}

		ticker := time.NewTicker(flushEvery)
		defer ticker.Stop()

		count := 0
		for {
			select {
			case task, ok := <-s.deleteCh:
				if !ok {
					return
				}
				if task.UserID == "" || len(task.IDs) == 0 {
					continue
				}
				set := pending[task.UserID]
				if set == nil {
					set = make(map[string]struct{})
					pending[task.UserID] = set
				}
				for _, id := range task.IDs {
					id = strings.TrimSpace(id)
					if id == "" {
						continue
					}
					set[id] = struct{}{}
					count++
				}
				if count >= batchSize {
					flush()
					count = 0
				}

			case <-ticker.C:
				if count > 0 {
					flush()
					count = 0
				}
			}
		}
	}()
}

// ResolveWithDeleted resolves a short id and reports whether it was deleted.
func (s *Shortener) ResolveWithDeleted(id string) (string, bool, bool) {
	if pg, ok := s.repo.(*repository.PostgresRepository); ok {
		return pg.GetWithDeleted(id)
	}

	original, ok := s.repo.Get(id)

	return original, ok, false
}

func (s *Shortener) Stats() (urls int, users int, err error) {
	st, ok := s.repo.(interface {
		CountURLs() (int, error)
		CountUsers() (int, error)
	})
	if !ok {
		return 0, 0, nil
	}

	u, err := st.CountURLs()
	if err != nil {
		return 0, 0, err
	}
	
	us, err := st.CountUsers()
	if err != nil {
		return 0, 0, err
	}

	return u, us, nil
}

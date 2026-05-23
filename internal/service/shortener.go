package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sastromikus/pip_shortener/internal/model"
)

const (
	idLen      = 8
	stringbase = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
	ErrEmptyURL          = errors.New("empty url")
	ErrUnsupportedScheme = errors.New("unsupported scheme")
	ErrEmptyHost         = errors.New("empty host")
	ErrGenerateID        = errors.New("could not generate unique id")
	ErrStorage           = errors.New("storage error")
)

type URLRepository interface {
	Get(id string) (string, bool)
	GetByOriginal(original string) (string, bool)
	PutIfAbsent(id string, original string) (bool, error)
	PutBatchIfAbsent(items []model.URLItem) error
}

type UserURLRepository interface {
	AddUserURL(userID, shortID string) error
	AddUserURLs(userID string, shortIDs []string) error
	ListUserURLs(userID string) ([]model.UserURL, error)
}

type DeletedURLRepository interface {
	GetWithDeleted(id string) (string, bool, bool)
	MarkDeleted(userID string, ids []string) error
}

type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

type BatchResult struct {
	CorrelationID string
	ID            string
}

type DeleteTask struct {
	UserID string
	IDs    []string
}

type Shortener struct {
	repo     URLRepository
	deleteCh chan DeleteTask
}

func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{
		repo:     repo,
		deleteCh: make(chan DeleteTask, 1024),
	}
}

func (s *Shortener) Shorten(raw string) (string, error) {
	id, _, err := s.ShortenWithExisting(raw)
	return id, err
}

func (s *Shortener) ShortenForUser(raw string, userID string) (string, bool, error) {
	id, existed, err := s.ShortenWithExisting(raw)
	if err != nil {
		return "", false, err
	}

	if err := s.addUserURL(userID, id); err != nil {
		return "", false, err
	}

	return id, existed, nil
}

func (s *Shortener) ShortenWithExisting(raw string) (string, bool, error) {
	normalized, err := normalizeURL(raw)
	if err != nil {
		return "", false, err
	}

	if existingID, ok := s.repo.GetByOriginal(normalized); ok {
		return existingID, true, nil
	}

	return s.shortenWithUniqueID(normalized, idLen, 10)
}

func (s *Shortener) ShortenBatch(items []BatchItem, userID string) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(items))
	normalizedByIndex := make([]string, 0, len(items))
	idByOriginal := make(map[string]string, len(items))
	usedIDs := make(map[string]struct{}, len(items))
	toCreate := make([]model.URLItem, 0, len(items))

	for _, item := range items {
		normalized, err := normalizeURL(item.OriginalURL)
		if err != nil {
			return nil, err
		}

		normalizedByIndex = append(normalizedByIndex, normalized)
		results = append(results, BatchResult{CorrelationID: item.CorrelationID})

		if id, ok := idByOriginal[normalized]; ok {
			results[len(results)-1].ID = id
			continue
		}

		if existingID, ok := s.repo.GetByOriginal(normalized); ok {
			idByOriginal[normalized] = existingID
			results[len(results)-1].ID = existingID
			continue
		}

		id, err := s.generateBatchID(usedIDs, idLen, 10)
		if err != nil {
			return nil, err
		}

		idByOriginal[normalized] = id
		results[len(results)-1].ID = id
		toCreate = append(toCreate, model.URLItem{ID: id, Original: normalized})
	}

	if len(toCreate) > 0 {
		if err := s.repo.PutBatchIfAbsent(toCreate); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorage, err)
		}
	}

	for _, item := range toCreate {
		if storedOriginal, ok := s.repo.Get(item.ID); ok && storedOriginal == item.Original {
			continue
		}

		existingID, ok := s.repo.GetByOriginal(item.Original)
		if !ok {
			return nil, ErrStorage
		}
		idByOriginal[item.Original] = existingID
	}

	shortIDs := make([]string, 0, len(results))
	for i, normalized := range normalizedByIndex {
		results[i].ID = idByOriginal[normalized]
		shortIDs = append(shortIDs, results[i].ID)
	}

	if err := s.addUserURLs(userID, shortIDs); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Shortener) Resolve(id string) (string, bool) {
	return s.repo.Get(id)
}

func (s *Shortener) ResolveWithDeleted(id string) (string, bool, bool) {
	store, ok := s.repo.(DeletedURLRepository)
	if ok {
		return store.GetWithDeleted(id)
	}

	original, found := s.repo.Get(id)
	return original, found, false
}

func (s *Shortener) ListUserURLs(userID string) ([]model.UserURL, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	store, ok := s.repo.(UserURLRepository)
	if !ok {
		return nil, nil
	}

	return store.ListUserURLs(userID)
}

func (s *Shortener) EnqueueDelete(userID string, ids []string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || len(ids) == 0 {
		return
	}

	select {
	case s.deleteCh <- DeleteTask{UserID: userID, IDs: ids}:
	default:
	}
}

func (s *Shortener) StartDeleteWorker(batchSize int, flushEvery time.Duration, logError func(error)) {
	deleter, ok := s.repo.(DeletedURLRepository)
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

				if err := deleter.MarkDeleted(userID, ids); err != nil && logError != nil {
					logError(err)
				}
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
					flush()
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

func (s *Shortener) addUserURL(userID string, shortID string) error {
	return s.addUserURLs(userID, []string{shortID})
}

func (s *Shortener) addUserURLs(userID string, shortIDs []string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || len(shortIDs) == 0 {
		return nil
	}

	store, ok := s.repo.(UserURLRepository)
	if !ok {
		return nil
	}

	if len(shortIDs) == 1 {
		if err := store.AddUserURL(userID, shortIDs[0]); err != nil {
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}
		return nil
	}

	if err := store.AddUserURLs(userID, shortIDs); err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return nil
}

func (s *Shortener) shortenWithUniqueID(raw string, length int, tries int) (string, bool, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", false, fmt.Errorf("%w: %v", ErrGenerateID, err)
		}

		created, err := s.repo.PutIfAbsent(id, raw)
		if err != nil {
			if existingID, ok := s.repo.GetByOriginal(raw); ok {
				return existingID, true, nil
			}
			return "", false, fmt.Errorf("%w: %v", ErrStorage, err)
		}

		if created {
			return id, false, nil
		}
	}

	if existingID, ok := s.repo.GetByOriginal(raw); ok {
		return existingID, true, nil
	}

	return "", false, ErrGenerateID
}

func (s *Shortener) generateBatchID(used map[string]struct{}, length int, tries int) (string, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrGenerateID, err)
		}

		if _, ok := used[id]; ok {
			continue
		}
		if _, ok := s.repo.Get(id); ok {
			continue
		}

		used[id] = struct{}{}
		return id, nil
	}

	return "", ErrGenerateID
}

func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrEmptyURL
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	if err := validateURL(raw); err != nil {
		return "", err
	}

	return raw, nil
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
		return ErrUnsupportedScheme
	}
	if u.Host == "" {
		return ErrEmptyHost
	}

	return nil
}

package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sastromikus/pip_shortener/internal/model"
)

const (
	idLen      = 8
	stringbase = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// ErrEmptyURL is returned when an empty URL is passed to the shortener.
var ErrEmptyURL = errors.New("empty url")

// ErrUnsupportedScheme is returned when a URL has an unsupported scheme.
var ErrUnsupportedScheme = errors.New("unsupported scheme")

// ErrEmptyHost is returned when a URL does not contain a host.
var ErrEmptyHost = errors.New("empty host")

// ErrGenerateID is returned when the service cannot generate a unique short ID.
var ErrGenerateID = errors.New("could not generate unique id")

// ErrStorage is returned when a storage operation fails.
var ErrStorage = errors.New("storage error")

// ErrDeleteQueueFull is returned when an asynchronous delete task cannot be queued.
var ErrDeleteQueueFull = errors.New("delete queue is full")

// URLRepository describes storage operations required by Shortener.
type URLRepository interface {
	Get(ctx context.Context, id string) (string, bool)
	GetWithDeleted(ctx context.Context, id string) (string, bool, bool)
	GetByOriginal(ctx context.Context, original string) (string, bool)
	PutIfAbsent(ctx context.Context, id string, original string) (bool, error)
	PutBatchIfAbsent(ctx context.Context, items []model.URLItem) error
	AddUserURL(ctx context.Context, userID, shortID string) error
	AddUserURLs(ctx context.Context, userID string, shortIDs []string) error
	ListUserURLs(ctx context.Context, userID string) ([]model.UserURL, error)
	MarkDeleted(ctx context.Context, userID string, ids []string) error
	Stats(ctx context.Context) (urls int, users int, err error)
}

// BatchItem describes one batch shortening request.
type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

// BatchResult describes one batch shortening result.
type BatchResult struct {
	CorrelationID string
	ID            string
}

// DeleteTask represents a request to delete multiple short URLs for a user.
type DeleteTask struct {
	UserID string
	IDs    []string
}

// Shortener implements URL shortening business logic.
type Shortener struct {
	repo     URLRepository
	deleteCh chan DeleteTask
}

// NewShortener creates a new Shortener using the provided repository.
func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{
		repo:     repo,
		deleteCh: make(chan DeleteTask, 1024),
	}
}

// Shorten creates or returns a short id for the provided URL.
func (s *Shortener) Shorten(raw string) (string, error) {
	id, _, err := s.ShortenWithExistingContext(context.Background(), raw)
	return id, err
}

// ShortenForUserContext creates a short URL using the provided context.
func (s *Shortener) ShortenForUserContext(ctx context.Context, raw string, userID string) (string, bool, error) {
	id, existed, err := s.ShortenWithExistingContext(ctx, raw)
	if err != nil {
		return "", false, err
	}

	if err := s.addUserURL(ctx, userID, id); err != nil {
		return "", false, err
	}

	return id, existed, nil
}

// ShortenWithExistingContext creates a short id or returns an existing one using context.
func (s *Shortener) ShortenWithExistingContext(ctx context.Context, raw string) (string, bool, error) {
	normalized, err := normalizeURL(raw)
	if err != nil {
		return "", false, err
	}

	if existingID, ok := s.repo.GetByOriginal(ctx, normalized); ok {
		return existingID, true, nil
	}

	return s.shortenWithUniqueID(ctx, normalized, idLen, 10)
}

// ShortenBatchContext shortens multiple URLs using context.
func (s *Shortener) ShortenBatchContext(ctx context.Context, items []BatchItem, userID string) ([]BatchResult, error) {
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

		if existingID, ok := s.repo.GetByOriginal(ctx, normalized); ok {
			idByOriginal[normalized] = existingID
			results[len(results)-1].ID = existingID
			continue
		}

		id, err := s.generateBatchID(ctx, usedIDs, idLen, 10)
		if err != nil {
			return nil, err
		}

		idByOriginal[normalized] = id
		results[len(results)-1].ID = id
		toCreate = append(toCreate, model.URLItem{ID: id, Original: normalized})
	}

	if len(toCreate) > 0 {
		if err := s.repo.PutBatchIfAbsent(ctx, toCreate); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrStorage, err)
		}
	}

	for _, item := range toCreate {
		if storedOriginal, ok := s.repo.Get(ctx, item.ID); ok && storedOriginal == item.Original {
			continue
		}

		existingID, ok := s.repo.GetByOriginal(ctx, item.Original)
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

	if err := s.addUserURLs(ctx, userID, shortIDs); err != nil {
		return nil, err
	}

	return results, nil
}

// ResolveContext returns the original URL by short id using context.
func (s *Shortener) ResolveContext(ctx context.Context, id string) (string, bool) {
	return s.repo.Get(ctx, id)
}

// ResolveWithDeleted resolves a short id and reports whether it was deleted.
func (s *Shortener) ResolveWithDeleted(ctx context.Context, id string) (string, bool, bool) {
	return s.repo.GetWithDeleted(ctx, id)
}

// ListUserURLs returns all URLs created by the given user.
func (s *Shortener) ListUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	return s.repo.ListUserURLs(ctx, userID)
}

// Stats returns total URL and user counts.
func (s *Shortener) Stats(ctx context.Context) (int, int, error) {
	urls, users, err := s.repo.Stats(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", ErrStorage, err)
	}

	return urls, users, nil
}

// EnqueueDelete queues an asynchronous delete request.
func (s *Shortener) EnqueueDelete(userID string, ids []string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || len(ids) == 0 {
		return nil
	}

	cleanIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			cleanIDs = append(cleanIDs, id)
		}
	}

	if len(cleanIDs) == 0 {
		return nil
	}

	select {
	case s.deleteCh <- DeleteTask{UserID: userID, IDs: cleanIDs}:
		return nil
	default:
		return ErrDeleteQueueFull
	}
}

// StartDeleteWorker starts background processing of delete tasks.
func (s *Shortener) StartDeleteWorker(ctx context.Context, batchSize int, flushEvery time.Duration, logError func(error)) func() {
	if batchSize <= 0 {
		batchSize = 64
	}
	if flushEvery <= 0 {
		flushEvery = 500 * time.Millisecond
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		pending := make(map[string]map[string]struct{})
		count := 0

		addTask := func(task DeleteTask) {
			if task.UserID == "" || len(task.IDs) == 0 {
				return
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
		}

		flush := func() {
			remaining := 0
			for userID, set := range pending {
				if len(set) == 0 {
					delete(pending, userID)
					continue
				}

				ids := make([]string, 0, len(set))
				for id := range set {
					ids = append(ids, id)
				}

				flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				err := s.repo.MarkDeleted(flushCtx, userID, ids)
				cancel()
				if err != nil {
					remaining += len(set)
					if logError != nil {
						logError(fmt.Errorf("mark URLs deleted for user %q: %w", userID, err))
					}
					continue
				}

				delete(pending, userID)
			}
			count = remaining
		}

		drainAndFlush := func() {
			for {
				select {
				case task := <-s.deleteCh:
					addTask(task)
				default:
					flush()
					return
				}
			}
		}

		ticker := time.NewTicker(flushEvery)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				drainAndFlush()
				return

			case task := <-s.deleteCh:
				addTask(task)
				if count >= batchSize {
					flush()
				}

			case <-ticker.C:
				if count > 0 {
					flush()
				}
			}
		}
	}()

	return wg.Wait
}

func (s *Shortener) addUserURL(ctx context.Context, userID string, shortID string) error {
	return s.addUserURLs(ctx, userID, []string{shortID})
}

func (s *Shortener) addUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || len(shortIDs) == 0 {
		return nil
	}

	if len(shortIDs) == 1 {
		if err := s.repo.AddUserURL(ctx, userID, shortIDs[0]); err != nil {
			return fmt.Errorf("%w: %w", ErrStorage, err)
		}
		return nil
	}

	if err := s.repo.AddUserURLs(ctx, userID, shortIDs); err != nil {
		return fmt.Errorf("%w: %w", ErrStorage, err)
	}
	return nil
}

func (s *Shortener) shortenWithUniqueID(ctx context.Context, raw string, length int, tries int) (string, bool, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", false, fmt.Errorf("%w: %w", ErrGenerateID, err)
		}

		created, err := s.repo.PutIfAbsent(ctx, id, raw)
		if err != nil {
			if existingID, ok := s.repo.GetByOriginal(ctx, raw); ok {
				return existingID, true, nil
			}
			return "", false, fmt.Errorf("%w: %w", ErrStorage, err)
		}

		if created {
			return id, false, nil
		}
	}

	if existingID, ok := s.repo.GetByOriginal(ctx, raw); ok {
		return existingID, true, nil
	}

	return "", false, ErrGenerateID
}

func (s *Shortener) generateBatchID(ctx context.Context, used map[string]struct{}, length int, tries int) (string, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrGenerateID, err)
		}

		if _, ok := used[id]; ok {
			continue
		}
		if _, ok := s.repo.Get(ctx, id); ok {
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

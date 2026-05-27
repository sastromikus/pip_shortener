package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

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

type UserURL struct {
	ID          string
	OriginalURL string
}

type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

type BatchResult struct {
	CorrelationID string
	ID            string
}

type Shortener struct {
	repo URLRepository

	mu       sync.Mutex
	userURLs map[string][]UserURL
}

func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{
		repo:     repo,
		userURLs: make(map[string][]UserURL),
	}
}

func (s *Shortener) Shorten(raw string) (string, error) {
	id, _, err := s.ShortenWithExisting(raw)
	return id, err
}

func (s *Shortener) ShortenForUser(raw string, userID string) (string, error) {
	id, _, err := s.ShortenWithExistingForUser(raw, userID)
	return id, err
}

func (s *Shortener) ShortenWithExisting(raw string) (string, bool, error) {
	normalized, err := normalizeURL(raw)
	if err != nil {
		return "", false, err
	}

	if existingID, ok := s.repo.GetByOriginal(normalized); ok {
		return existingID, true, nil
	}

	id, existed, err := s.shortenWithUniqueID(normalized, idLen, 10)
	if err != nil {
		return "", false, err
	}

	return id, existed, nil
}

func (s *Shortener) ShortenWithExistingForUser(raw string, userID string) (string, bool, error) {
	normalized, err := normalizeURL(raw)
	if err != nil {
		return "", false, err
	}

	id, existed, err := s.shortenNormalizedWithExisting(normalized)
	if err != nil {
		return "", false, err
	}

	s.addUserURL(userID, id, normalized)

	return id, existed, nil
}

func (s *Shortener) ShortenBatch(items []BatchItem) ([]BatchResult, error) {
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
		results = append(results, BatchResult{
			CorrelationID: item.CorrelationID,
		})

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

		toCreate = append(toCreate, model.URLItem{
			ID:       id,
			Original: normalized,
		})
	}

	if len(toCreate) == 0 {
		return results, nil
	}

	if err := s.repo.PutBatchIfAbsent(toCreate); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorage, err)
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

	for i, normalized := range normalizedByIndex {
		results[i].ID = idByOriginal[normalized]
	}

	return results, nil
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

func (s *Shortener) UserURLs(userID string) []UserURL {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.userURLs[userID]
	if len(items) == 0 {
		return nil
	}

	out := make([]UserURL, len(items))
	copy(out, items)
	return out
}

func (s *Shortener) Resolve(id string) (string, bool) {
	return s.repo.Get(id)
}

func (s *Shortener) shortenNormalizedWithExisting(normalized string) (string, bool, error) {
	if existingID, ok := s.repo.GetByOriginal(normalized); ok {
		return existingID, true, nil
	}

	return s.shortenWithUniqueID(normalized, idLen, 10)
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

func (s *Shortener) addUserURL(userID string, id string, original string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.userURLs[userID] = append(s.userURLs[userID], UserURL{
		ID:          id,
		OriginalURL: original,
	})
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

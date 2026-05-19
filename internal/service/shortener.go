package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
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

	for _, item := range items {
		id, _, err := s.ShortenWithExisting(item.OriginalURL)
		if err != nil {
			return nil, err
		}

		results = append(results, BatchResult{
			CorrelationID: item.CorrelationID,
			ID:            id,
		})
	}

	return results, nil
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

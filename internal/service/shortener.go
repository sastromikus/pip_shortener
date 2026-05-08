package service

import (
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"sync"
)

const (
	idLen      = 8
	stringbase = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

type Shortener struct {
	repo URLRepository

	mu       sync.RWMutex
	userURLs map[string][]UserURL
}

type UserURL struct {
	ID          string
	OriginalURL string
}

func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{
		repo:     repo,
		userURLs: make(map[string][]UserURL),
	}
}

func (s *Shortener) Shorten(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty url")
	}

	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	if err := validateURL(raw); err != nil {
		return "", err
	}

	id, err := s.generateUniqueID(idLen, 10)
	if err != nil {
		return "", err
	}

	s.repo.Put(id, raw)
	return id, nil
}

func (s *Shortener) ShortenForUser(raw string, userID string) (string, error) {
	id, err := s.Shorten(raw)
	if err != nil {
		return "", err
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return id, nil
	}

	original, ok := s.Resolve(id)
	if !ok {
		return id, nil
	}

	s.mu.Lock()
	s.userURLs[userID] = append(s.userURLs[userID], UserURL{
		ID:          id,
		OriginalURL: original,
	})
	s.mu.Unlock()

	return id, nil
}

func (s *Shortener) UserURLs(userID string) []UserURL {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

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

package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
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
)

type URLRepository interface {
	Get(id string) (string, bool)
	PutIfAbsent(id string, original string) bool
}

type Shortener struct {
	repo URLRepository
}

func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{repo: repo}
}

func (s *Shortener) Shorten(raw string) (string, error) {
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

	id, err := s.shortenWithUniqueID(raw, idLen, 10)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *Shortener) Resolve(id string) (string, bool) {
	return s.repo.Get(id)
}

func (s *Shortener) shortenWithUniqueID(raw string, length int, tries int) (string, error) {
	for i := 0; i < tries; i++ {
		id, err := randomstringbase(length)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrGenerateID, err)
		}

		if s.repo.PutIfAbsent(id, raw) {
			return id, nil
		}
	}

	return "", ErrGenerateID
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

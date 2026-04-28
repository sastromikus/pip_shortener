package service

import (
	"crypto/rand"
	"errors"
	"net/url"
	"strings"

	"github.com/sastromikus/pip_shortener/internal/repository"
)

const (
	idLen  = 8
	stringbase = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

type Shortener struct {
	repo URLRepository
}

func NewShortener(repo URLRepository) *Shortener {
	return &Shortener{repo: repo}
}

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
package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
	"github.com/sirupsen/logrus"
)

func BenchmarkPOST_Shorten_TextPlain(b *testing.B) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(io.Discard)

	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	u := mustURL("http://localhost:8080/")
	body := []byte("http://example.com/path")
	cookie := validUserCookie("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &http.Request{
			Method: http.MethodPost,
			URL:    u,
			Header: make(http.Header),
			Body:   io.NopCloser(bytes.NewReader(body)),
		}
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Cookie", cookie)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		_ = w.Result().Body.Close()
	}
}

func BenchmarkPOST_API_Shorten_JSON(b *testing.B) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(io.Discard)

	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	u := mustURL("http://localhost:8080/api/shorten")
	payload := []byte(`{"url":"https://practicum.yandex.ru/"}`)
	cookie := validUserCookie("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &http.Request{
			Method: http.MethodPost,
			URL:    u,
			Header: make(http.Header),
			Body:   io.NopCloser(bytes.NewReader(payload)),
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", cookie)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		_ = w.Result().Body.Close()
	}
}

func BenchmarkGET_Follow(b *testing.B) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetOutput(io.Discard)

	repo.Put("TESTID12", "https://example.com/path")
	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	u := mustURL("http://localhost:8080/TESTID12")
	cookie := validUserCookie("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    u,
			Header: make(http.Header),
			Body:   http.NoBody,
		}
		req.Header.Set("Cookie", cookie)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		_ = w.Result().Body.Close()
	}
}

func mustURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func validUserCookie(uid string) string {
	secret := os.Getenv("COOKIE_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(uid))
	sig := hex.EncodeToString(mac.Sum(nil))

	// ровно как в middleware: "user_id=uid:sig"
	return "user_id=" + uid + ":" + sig
}

package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
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

	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	body := strings.Repeat("http://example.com/path/", 8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
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

	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	payload := []byte(`{"url":"https://practicum.yandex.ru/"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/shorten", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
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

	repo.Put("TESTID12", "https://example.com/path")
	h := NewRouter(svc, "http://localhost:8080", logger, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/TESTID12", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		_ = w.Result().Body.Close()
	}
}
package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func TestPOST_Shorten_Returns201AndShortURL(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)
	h := NewRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader("https://practicum.yandex.ru/"))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, res.StatusCode)
	}

	ct := res.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(ct), "text/plain") {
		t.Fatalf("expected Content-Type text/plain, got %q", ct)
	}

	b, _ := io.ReadAll(res.Body)
	shortURL := strings.TrimSpace(string(b))

	if !strings.HasPrefix(shortURL, "http://localhost:8080/") {
		t.Fatalf("expected short URL with localhost:8080 prefix, got %q", shortURL)
	}

	id := strings.TrimPrefix(shortURL, "http://localhost:8080/")
	if id == "" {
		t.Fatalf("expected non-empty id in short url, got %q", shortURL)
	}

	if _, ok := repo.Get(id); !ok {
		t.Fatalf("expected id %q to be stored", id)
	}
}

func TestGET_Redirect_Returns307AndLocation(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)
	h := NewRouter(svc)

	const id = "TESTID12"
	const original = "https://example.com/path"
	repo.Put(id, original)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/"+id, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, res.StatusCode)
	}
	if loc := res.Header.Get("Location"); loc != original {
		t.Fatalf("expected Location %q, got %q", original, loc)
	}
}

func TestInvalidRequests_Return400(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)
	h := NewRouter(svc)

	tests := []struct {
		name string
		req  *http.Request
	}{
		{
			name: "GET / (no id)",
			req:  httptest.NewRequest(http.MethodGet, "http://localhost:8080/", nil),
		},
		{
			name: "POST wrong content-type",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader("https://ya.ru"))
				r.Header.Set("Content-Type", "application/json")
				return r
			}(),
		},
		{
			name: "POST empty body",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "http://localhost:8080/", strings.NewReader(""))
				r.Header.Set("Content-Type", "text/plain")
				return r
			}(),
		},
		{
			name: "GET unknown id",
			req:  httptest.NewRequest(http.MethodGet, "http://localhost:8080/NO_SUCH_ID", nil),
		},
		{
			name: "GET extra path segment",
			req:  httptest.NewRequest(http.MethodGet, "http://localhost:8080/a/b", nil),
		},
		{
			name: "unknown method",
			req:  httptest.NewRequest(http.MethodPut, "http://localhost:8080/", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, tt.req)

			res := w.Result()
			res.Body.Close()

			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", res.StatusCode)
			}
		})
	}
}
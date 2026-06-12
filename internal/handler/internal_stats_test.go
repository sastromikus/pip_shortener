package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func newStatsRouter(t *testing.T, trustedSubnet string) http.Handler {
	t.Helper()

	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)
	return NewRouter(svc, "http://localhost:8080", nil, (*sql.DB)(nil), nil, trustedSubnet)
}

func TestInternalStats_ForbiddenWhenNoSubnet(t *testing.T) {
	router := newStatsRouter(t, "")
	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestInternalStats_ForbiddenWhenBadCIDR(t *testing.T) {
	router := newStatsRouter(t, "not-a-cidr")
	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

func TestInternalStats_ForbiddenWhenIPMissingOrOutside(t *testing.T) {
	router := newStatsRouter(t, "127.0.0.0/8")

	for _, ip := range []string{"", "10.1.2.3"} {
		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		if ip != "" {
			req.Header.Set("X-Real-IP", ip)
		}
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("ip %q: want 403, got %d", ip, w.Code)
		}
	}
}

func TestInternalStats_OKReturnsJSONCounts(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	_, _, err := svc.ShortenForUserContext(context.Background(), "https://example.com/a", "u1")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}
	_, _, _ = svc.ShortenForUserContext(context.Background(), "https://example.com/b", "u1")
	_, _, _ = svc.ShortenForUserContext(context.Background(), "https://example.com/c", "u2")

	router := NewRouter(svc, "http://localhost:8080", nil, nil, nil, "127.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var out statsResponse
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.URLs != 3 {
		t.Fatalf("urls: want 3, got %d", out.URLs)
	}
	if out.Users != 2 {
		t.Fatalf("users: want 2, got %d", out.Users)
	}
}

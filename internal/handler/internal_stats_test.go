package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
)

func TestInternalStats_ForbiddenWhenNoSubnet(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	handleInternalStats(svc, "", w, req)

	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Result().StatusCode)
	}
}

func TestInternalStats_ForbiddenWhenBadCIDR(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	handleInternalStats(svc, "not-a-cidr", w, req)

	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", w.Result().StatusCode)
	}
}

func TestInternalStats_ForbiddenWhenIPMissingOrOutside(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/internal/stats", nil)
	w := httptest.NewRecorder()
	handleInternalStats(svc, "127.0.0.0/8", w, req)
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("missing ip: want 403, got %d", w.Result().StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/internal/stats", nil)
	req2.Header.Set("X-Real-IP", "10.1.2.3")
	w2 := httptest.NewRecorder()
	handleInternalStats(svc, "127.0.0.0/8", w2, req2)
	if w2.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("outside subnet: want 403, got %d", w2.Result().StatusCode)
	}
}

func TestInternalStats_OKReturnsJSONCounts(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo)

	id1, _, err := svc.ShortenForUser("https://example.com/a", "u1")
	if err != nil {
		t.Fatalf("shorten: %v", err)
	}
	_, _, _ = svc.ShortenForUser("https://example.com/b", "u1")
	_, _, _ = svc.ShortenForUser("https://example.com/c", "u2")

	if _, ok := repo.Get(id1); !ok {
		t.Fatalf("expected stored id %q", id1)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/internal/stats", nil)
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()

	handleInternalStats(svc, "127.0.0.0/8", w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}

	var out struct {
		URLs  int `json:"urls"`
		Users int `json:"users"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if out.URLs != 3 {
		t.Fatalf("urls: want 3, got %d", out.URLs)
	}
	if out.Users != 2 {
		t.Fatalf("users: want 2, got %d", out.Users)
	}
}
package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignVerifyCookie_RoundTrip(t *testing.T) {
	secret := []byte("secret")
	uid := "user123"

	s := signCookie(uid, secret)
	gotUID, ok := verifyCookie(s, secret)
	if !ok || gotUID != uid {
		t.Fatalf("verify: want (%q,true), got (%q,%v)", uid, gotUID, ok)
	}

	tampered := s + "00"
	_, ok = verifyCookie(tampered, secret)
	if ok {
		t.Fatalf("tampered must not verify")
	}
}

func TestAuth_SetsCookieWhenMissing(t *testing.T) {
	logger := io.Discard

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok || uid == "" {
			t.Fatalf("expected user id in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Auth()(h)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	rec := httptest.NewRecorder()

	_ = logger
	wrapped.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}
	if res.Header.Get("Set-Cookie") == "" {
		t.Fatalf("expected Set-Cookie header")
	}
}

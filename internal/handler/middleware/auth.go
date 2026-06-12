package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	sharedauth "github.com/sastromikus/pip_shortener/internal/auth"
)

const (
	cookieName = "user_id"
)

type ctxKey int

const (
	ctxUserIDKey ctxKey = iota
	ctxBadCookieKey
)

// UserIDFromContext extracts a verified user id from request context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserIDKey)
	s, ok := v.(string)

	return s, ok
}

// BadCookieNoID reports whether the request contained an invalid user cookie.
func BadCookieNoID(ctx context.Context) bool {
	v := ctx.Value(ctxBadCookieKey)
	b, _ := v.(bool)

	return b
}

// Auth ensures a signed user_id cookie exists and stores user id in request context.
func Auth() func(http.Handler) http.Handler {
	secret := sharedauth.Secret()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err == nil {
				val := strings.TrimSpace(c.Value)
				if val == "" {
					ctx := context.WithValue(r.Context(), ctxBadCookieKey, true)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				uid, ok := verifyCookie(val, secret)
				if ok && uid != "" {
					ctx := context.WithValue(r.Context(), ctxUserIDKey, uid)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				ctx := context.WithValue(r.Context(), ctxBadCookieKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			newUID := newUserID()
			http.SetCookie(w, buildCookie(newUID, secret))
			ctx := context.WithValue(r.Context(), ctxUserIDKey, newUID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func buildCookie(uid string, secret []byte) *http.Cookie {
	val := signCookie(uid, secret)

	return &http.Cookie{
		Name:     cookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	}
}

func signCookie(uid string, secret []byte) string {
	return sharedauth.Sign(uid, secret)
}

func verifyCookie(val string, secret []byte) (string, bool) {
	return sharedauth.Verify(val, secret)
}

func newUserID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

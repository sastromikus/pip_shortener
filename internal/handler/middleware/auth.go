package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	cookieName = "user_id"
)

type ctxKey int

const (
	ctxUserIDKey ctxKey = iota
	ctxBadCookieKey
)

func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserIDKey)
	s, _ := v.(string)

	return s, s != ""
}

func BadCookieNoID(ctx context.Context) bool {
	v := ctx.Value(ctxBadCookieKey)
	b, _ := v.(bool)

	return b
}

func Auth() func(http.Handler) http.Handler {
	secret := []byte(os.Getenv("COOKIE_SECRET"))
	if len(secret) == 0 {
		secret = []byte("dev-secret")
	}

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

				newUID := newUserID()
				http.SetCookie(w, buildCookie(newUID, secret))
				ctx := context.WithValue(r.Context(), ctxUserIDKey, newUID)
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
		Expires: time.Now().Add(365 * 24 * time.Hour),
	}
}

func signCookie(uid string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(uid))
	sum := mac.Sum(nil)

	out := make([]byte, 0, len(uid)+1+hex.EncodedLen(len(sum)))
	out = append(out, uid...)
	out = append(out, ':')

	dst := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(dst, sum)
	out = append(out, dst...)

	return string(out)
}

func verifyCookie(val string, secret []byte) (string, bool) {
	uid, sigHex, ok := strings.Cut(val, ":")
	if !ok || uid == "" || sigHex == "" {
		return "", false
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", false
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(uid))
	sum := mac.Sum(nil)

	if !hmac.Equal(sum, sig) {
		return "", false
	}
	return uid, true
}

func newUserID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	
	return hex.EncodeToString(b)
}
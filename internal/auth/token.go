package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

const defaultSecret = "dev-secret"

// Secret returns the shared secret used by both HTTP cookies and gRPC metadata.
func Secret() []byte {
	value, ok := os.LookupEnv("COOKIE_SECRET")
	if !ok || value == "" {
		value = defaultSecret
	}
	return []byte(value)
}

// Sign creates a signed user token in the form userID:hex(HMAC-SHA256(userID)).
func Sign(userID string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(userID))
	sum := mac.Sum(nil)

	return userID + ":" + hex.EncodeToString(sum)
}

// Verify validates a signed user token and returns its user ID.
func Verify(value string, secret []byte) (string, bool) {
	userID, signatureHex, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || userID == "" || signatureHex == "" {
		return "", false
	}

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", false
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(userID))
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, signature) {
		return "", false
	}

	return userID, true
}

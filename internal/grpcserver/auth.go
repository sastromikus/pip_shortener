package grpcserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"google.golang.org/grpc/metadata"
)

var cookieSecret = []byte("change-me-secret")

func UserIDFromAuthMeta(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", false
	}
	return verify(vals[0])
}

func verify(v string) (string, bool) {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return "", false
	}
	uid := parts[0]
	sigHex := parts[1]
	if uid == "" || sigHex == "" {
		return "", false
	}

	mac := hmac.New(sha256.New, cookieSecret)
	_, _ = mac.Write([]byte(uid))
	want := hex.EncodeToString(mac.Sum(nil))

	return uid, hmac.Equal([]byte(want), []byte(sigHex))
}

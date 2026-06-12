package grpcserver

import (
	"context"
	"strings"

	sharedauth "github.com/sastromikus/pip_shortener/internal/auth"
	"google.golang.org/grpc/metadata"
)

// UserIDFromAuthMeta reads and validates the authorization metadata value.
// Both the raw signed token and the conventional "Bearer <token>" form are accepted.
func UserIDFromAuthMeta(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", false
	}

	value := strings.TrimSpace(values[0])
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		value = strings.TrimSpace(value[len("bearer "):])
	}

	return sharedauth.Verify(value, sharedauth.Secret())
}

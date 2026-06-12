package grpcserver

import (
	"context"
	"strings"
	"testing"

	shortenerv1 "github.com/sastromikus/pip_shortener/api/grpc"
	sharedauth "github.com/sastromikus/pip_shortener/internal/auth"
	"github.com/sastromikus/pip_shortener/internal/repository"
	"github.com/sastromikus/pip_shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	return New(service.NewShortener(repository.NewMemoryRepository()), "http://localhost:8080")
}

func authorizedContext(t *testing.T, userID string) context.Context {
	t.Helper()

	token := sharedauth.Sign(userID, sharedauth.Secret())
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", token))
}

func TestServerShortenURLRequiresAuthorization(t *testing.T) {
	server := newTestServer(t)
	req := shortenerv1.URLShortenRequest_builder{
		Url: "https://example.com",
	}.Build()

	_, err := server.ShortenURL(context.Background(), req)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestServerShortenAndExpandURL(t *testing.T) {
	server := newTestServer(t)
	shortenReq := shortenerv1.URLShortenRequest_builder{
		Url: "https://example.com/path",
	}.Build()

	shortenResp, err := server.ShortenURL(authorizedContext(t, "user-1"), shortenReq)
	if err != nil {
		t.Fatalf("ShortenURL: %v", err)
	}
	if !strings.HasPrefix(shortenResp.GetResult(), "http://localhost:8080/") {
		t.Fatalf("unexpected short URL %q", shortenResp.GetResult())
	}

	id := strings.TrimPrefix(shortenResp.GetResult(), "http://localhost:8080/")
	expandReq := shortenerv1.URLExpandRequest_builder{
		Id: id,
	}.Build()
	expandResp, err := server.ExpandURL(context.Background(), expandReq)
	if err != nil {
		t.Fatalf("anonymous ExpandURL: %v", err)
	}
	if expandResp.GetResult() != "https://example.com/path" {
		t.Fatalf("want original URL, got %q", expandResp.GetResult())
	}
}

func TestServerListUserURLs(t *testing.T) {
	server := newTestServer(t)
	ctx := authorizedContext(t, "user-1")
	request := shortenerv1.UserURLsRequest_builder{}.Build()

	empty, err := server.ListUserURLs(ctx, request)
	if err != nil {
		t.Fatalf("empty ListUserURLs: %v", err)
	}
	if len(empty.GetUrl()) != 0 {
		t.Fatalf("want empty list, got %d items", len(empty.GetUrl()))
	}

	for _, rawURL := range []string{"https://example.com/a", "https://example.com/b"} {
		req := shortenerv1.URLShortenRequest_builder{
			Url: rawURL,
		}.Build()
		if _, err := server.ShortenURL(ctx, req); err != nil {
			t.Fatalf("ShortenURL(%q): %v", rawURL, err)
		}
	}

	response, err := server.ListUserURLs(ctx, request)
	if err != nil {
		t.Fatalf("ListUserURLs: %v", err)
	}
	if len(response.GetUrl()) != 2 {
		t.Fatalf("want 2 items, got %d", len(response.GetUrl()))
	}
	for _, item := range response.GetUrl() {
		if item.GetShortUrl() == "" || item.GetOriginalUrl() == "" {
			t.Fatalf("incomplete URL item: %v", item)
		}
	}
}

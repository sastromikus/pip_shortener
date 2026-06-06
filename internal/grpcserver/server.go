package grpcserver

import (
	"context"
	"strings"

	shortenerv1 "github.com/sastromikus/pip_shortener/api/grpc"
	"github.com/sastromikus/pip_shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server implements the URL shortener gRPC service.
type Server struct {
	shortenerv1.UnimplementedShortenerServiceServer

	svc     *service.Shortener
	baseURL string
}

// New creates a gRPC server backed by the shared shortener service.
func New(svc *service.Shortener, baseURL string) *Server {
	return &Server{
		svc:     svc,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// ShortenURL creates a short URL for the authenticated user.
func (s *Server) ShortenURL(ctx context.Context, req *shortenerv1.URLShortenRequest) (*shortenerv1.URLShortenResponse, error) {
	userID, ok := UserIDFromAuthMeta(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	raw := strings.TrimSpace(req.GetUrl())
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	id, _, err := s.svc.ShortenForUserContext(ctx, raw, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot shorten URL")
	}

	return &shortenerv1.URLShortenResponse{Result: s.baseURL + "/" + id}, nil
}

// ExpandURL resolves a short identifier to its original URL.
func (s *Server) ExpandURL(ctx context.Context, req *shortenerv1.URLExpandRequest) (*shortenerv1.URLExpandResponse, error) {
	if _, ok := UserIDFromAuthMeta(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	original, found, deleted := s.svc.ResolveWithDeleted(ctx, id)
	if !found {
		return nil, status.Error(codes.NotFound, "short URL not found")
	}
	if deleted {
		return nil, status.Error(codes.FailedPrecondition, "short URL was deleted")
	}

	return &shortenerv1.URLExpandResponse{Result: original}, nil
}

// ListUserURLs returns all URLs created by the authenticated user.
func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*shortenerv1.UserURLsResponse, error) {
	userID, ok := UserIDFromAuthMeta(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}

	items, err := s.svc.ListUserURLs(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot list user URLs")
	}

	out := make([]*shortenerv1.URLData, 0, len(items))
	for _, item := range items {
		out = append(out, &shortenerv1.URLData{
			ShortUrl:    s.baseURL + "/" + item.ShortID,
			OriginalUrl: item.Original,
		})
	}

	return &shortenerv1.UserURLsResponse{Url: out}, nil
}

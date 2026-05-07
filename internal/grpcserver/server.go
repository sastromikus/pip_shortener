package grpcserver

import (
	"context"
	"strings"

	"github.com/sastromikus/pip_shortener/api/grpc/shortenerv1"
	"github.com/sastromikus/pip_shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	shortenerv1.UnimplementedShortenerServiceServer

	svc     *service.Shortener
	baseURL string
}

func New(svc *service.Shortener, baseURL string) *Server {
	return &Server{
		svc:     svc,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *Server) ShortenURL(ctx context.Context, req *shortenerv1.URLShortenRequest) (*shortenerv1.URLShortenResponse, error) {
	userID, ok := UserIDFromAuthMeta(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing/invalid authorization")
	}

	raw := strings.TrimSpace(req.GetUrl())
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "empty url")
	}

	id, _, err := s.svc.ShortenForUser(raw, userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot shorten")
	}

	return &shortenerv1.URLShortenResponse{Result: s.baseURL + "/" + id}, nil
}

func (s *Server) ExpandURL(ctx context.Context, req *shortenerv1.URLExpandRequest) (*shortenerv1.URLExpandResponse, error) {
	_, ok := UserIDFromAuthMeta(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing/invalid authorization")
	}

	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id")
	}

	original, ok2, deleted := s.svc.ResolveWithDeleted(id)
	if !ok2 {
		return nil, status.Error(codes.NotFound, "not found")
	}
	if deleted {
		return nil, status.Error(codes.FailedPrecondition, "deleted")
	}

	return &shortenerv1.URLExpandResponse{Result: original}, nil
}

func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*shortenerv1.UserURLsResponse, error) {
	userID, ok := UserIDFromAuthMeta(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing/invalid authorization")
	}

	items, err := s.svc.ListUserURLs(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "storage error")
	}
	if len(items) == 0 {
		return &shortenerv1.UserURLsResponse{}, nil
	}

	out := make([]*shortenerv1.URLData, 0, len(items))
	for _, it := range items {
		out = append(out, &shortenerv1.URLData{
			ShortUrl:    s.baseURL + "/" + it.ShortID,
			OriginalUrl: it.Original,
		})
	}

	return &shortenerv1.UserURLsResponse{Url: out}, nil
}
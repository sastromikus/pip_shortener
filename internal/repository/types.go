package repository

import (
	"context"

	"github.com/sastromikus/pip_shortener/internal/model"
)

// UserURL is a compatibility alias for user URL records stored in model package.
type UserURL = model.UserURL

// URLRepository defines the storage operations required by the shortener service.
type URLRepository interface {
	Get(ctx context.Context, id string) (string, bool)
	GetWithDeleted(ctx context.Context, id string) (string, bool, bool)
	GetByOriginal(ctx context.Context, original string) (string, bool)
	PutIfAbsent(ctx context.Context, id string, original string) (bool, error)
	PutBatchIfAbsent(ctx context.Context, items []model.URLItem) error
	AddUserURL(ctx context.Context, userID, shortID string) error
	AddUserURLs(ctx context.Context, userID string, shortIDs []string) error
	ListUserURLs(ctx context.Context, userID string) ([]model.UserURL, error)
	MarkDeleted(ctx context.Context, userID string, ids []string) error
}

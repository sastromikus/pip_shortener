package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/sastromikus/pip_shortener/internal/model"
)

// FileRepository stores URLs in memory and persists them to files.
type FileRepository struct {
	path        string
	usersPath   string
	deletedPath string
	mem         *MemoryRepository
}

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// NewFileRepository creates a file-backed repository.
func NewFileRepository(path string) (*FileRepository, error) {
	if path == "" {
		return nil, errors.New("empty file storage path")
	}

	r := &FileRepository{
		path:        path,
		usersPath:   path + ".users",
		deletedPath: path + ".deleted",
		mem:         NewMemoryRepository(),
	}

	if err := r.load(); err != nil {
		return nil, fmt.Errorf("load file storage: %w", err)
	}
	if err := r.loadUsers(); err != nil {
		return nil, fmt.Errorf("load user file storage: %w", err)
	}
	if err := r.loadDeleted(); err != nil {
		return nil, fmt.Errorf("load deleted file storage: %w", err)
	}

	return r, nil
}

// Get returns the original URL by short id.
func (r *FileRepository) Get(ctx context.Context, id string) (string, bool) {
	return r.mem.Get(ctx, id)
}

// GetWithDeleted returns the original URL and deletion state by short id.
func (r *FileRepository) GetWithDeleted(ctx context.Context, id string) (string, bool, bool) {
	return r.mem.GetWithDeleted(ctx, id)
}

// GetByOriginal returns the short id for an original URL.
func (r *FileRepository) GetByOriginal(ctx context.Context, original string) (string, bool) {
	return r.mem.GetByOriginal(ctx, original)
}

// PutIfAbsent stores a URL only when neither short id nor original URL exists.
func (r *FileRepository) PutIfAbsent(ctx context.Context, id string, original string) (bool, error) {
	created, err := r.mem.PutIfAbsent(ctx, id, original)
	if err != nil {
		return false, err
	}
	if !created {
		return false, nil
	}

	if err := r.save(); err != nil {
		r.mem.Delete(id)
		return false, err
	}

	return true, nil
}

// PutBatchIfAbsent stores multiple URL records when they are absent.
func (r *FileRepository) PutBatchIfAbsent(ctx context.Context, items []model.URLItem) error {
	createdIDs := make([]string, 0, len(items))

	for _, item := range items {
		created, err := r.mem.PutIfAbsent(ctx, item.ID, item.Original)
		if err != nil {
			return err
		}
		if created {
			createdIDs = append(createdIDs, item.ID)
		}
	}

	if len(createdIDs) == 0 {
		return nil
	}

	if err := r.save(); err != nil {
		for _, id := range createdIDs {
			r.mem.Delete(id)
		}
		return err
	}

	return nil
}

// Put is kept for tests and simple compatibility. Business code should prefer PutIfAbsent.
func (r *FileRepository) Put(id string, original string) {
	_, _ = r.PutIfAbsent(context.Background(), id, original)
}

// Exists reports whether a short id is already stored.
func (r *FileRepository) Exists(id string) bool {
	_, ok := r.mem.Get(context.Background(), id)
	return ok
}

// AddUserURL associates a short id with a user.
func (r *FileRepository) AddUserURL(ctx context.Context, userID, shortID string) error {
	return r.AddUserURLs(ctx, userID, []string{shortID})
}

// AddUserURLs associates multiple short ids with a user.
func (r *FileRepository) AddUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if err := r.mem.AddUserURLs(ctx, userID, shortIDs); err != nil {
		return err
	}

	return r.saveUsers()
}

// ListUserURLs returns all non-deleted URLs owned by a user.
func (r *FileRepository) ListUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	return r.mem.ListUserURLs(ctx, userID)
}

// MarkDeleted marks user-owned URLs as deleted.
func (r *FileRepository) MarkDeleted(ctx context.Context, userID string, ids []string) error {
	if err := r.mem.MarkDeleted(ctx, userID, ids); err != nil {
		return err
	}

	return r.saveDeleted()
}

func (r *FileRepository) load() error {
	b, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}

	var recs []fileRecord
	if err := json.Unmarshal(b, &recs); err != nil {
		return err
	}

	for _, rec := range recs {
		_, _ = r.mem.PutIfAbsent(context.Background(), rec.ShortURL, rec.OriginalURL)
	}

	return nil
}

func (r *FileRepository) save() error {
	if err := ensureParentDir(r.path); err != nil {
		return err
	}

	items := r.mem.Items()
	keys := make([]string, 0, len(items))
	for id := range items {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	records := make([]fileRecord, 0, len(items))
	for i, id := range keys {
		records = append(records, fileRecord{
			UUID:        strconv.Itoa(i + 1),
			ShortURL:    id,
			OriginalURL: items[id],
		})
	}

	return writeJSONAtomic(r.path, records)
}

func (r *FileRepository) loadUsers() error {
	b, err := os.ReadFile(r.usersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}

	var items map[string][]string
	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}

	for userID, ids := range items {
		for _, id := range ids {
			r.mem.LoadUserURL(userID, id)
		}
	}

	return nil
}

func (r *FileRepository) saveUsers() error {
	if err := ensureParentDir(r.usersPath); err != nil {
		return err
	}

	return writeJSONAtomic(r.usersPath, r.mem.UserItems())
}

func (r *FileRepository) loadDeleted() error {
	b, err := os.ReadFile(r.deletedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(b, &ids); err != nil {
		var m map[string]bool
		if err2 := json.Unmarshal(b, &m); err2 != nil {
			return err
		}
		for id, deleted := range m {
			if deleted {
				r.mem.LoadDeleted(id)
			}
		}
		return nil
	}

	for _, id := range ids {
		r.mem.LoadDeleted(id)
	}

	return nil
}

func (r *FileRepository) saveDeleted() error {
	if err := ensureParentDir(r.deletedPath); err != nil {
		return err
	}

	return writeJSONAtomic(r.deletedPath, r.mem.DeletedItems())
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err == nil {
		return nil
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, path)
}

// Stats returns the number of stored URLs and users with at least one URL.
func (r *FileRepository) Stats(ctx context.Context) (int, int, error) {
	return r.mem.Stats(ctx)
}

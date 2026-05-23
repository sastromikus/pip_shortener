package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/sastromikus/pip_shortener/internal/model"
)

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

func (r *FileRepository) Get(id string) (string, bool) {
	return r.mem.Get(id)
}

func (r *FileRepository) GetWithDeleted(id string) (string, bool, bool) {
	return r.mem.GetWithDeleted(id)
}

func (r *FileRepository) GetByOriginal(original string) (string, bool) {
	return r.mem.GetByOriginal(original)
}

func (r *FileRepository) PutIfAbsent(id string, original string) (bool, error) {
	created, err := r.mem.PutIfAbsent(id, original)
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

func (r *FileRepository) PutBatchIfAbsent(items []model.URLItem) error {
	createdIDs := make([]string, 0, len(items))

	for _, item := range items {
		created, err := r.mem.PutIfAbsent(item.ID, item.Original)
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
	_, _ = r.PutIfAbsent(id, original)
}

func (r *FileRepository) Exists(id string) bool {
	_, ok := r.mem.Get(id)
	return ok
}

func (r *FileRepository) AddUserURL(userID, shortID string) error {
	return r.AddUserURLs(userID, []string{shortID})
}

func (r *FileRepository) AddUserURLs(userID string, shortIDs []string) error {
	if err := r.mem.AddUserURLs(userID, shortIDs); err != nil {
		return err
	}

	return r.saveUsers()
}

func (r *FileRepository) ListUserURLs(userID string) ([]model.UserURL, error) {
	return r.mem.ListUserURLs(userID)
}

func (r *FileRepository) MarkDeleted(userID string, ids []string) error {
	if err := r.mem.MarkDeleted(userID, ids); err != nil {
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
		_, _ = r.mem.PutIfAbsent(rec.ShortURL, rec.OriginalURL)
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

	return os.Rename(tmp, path)
}

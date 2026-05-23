package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sastromikus/pip_shortener/internal/model"
)

type FileRepository struct {
	mu          sync.Mutex
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
		return nil, err
	}

	if err := r.loadUsers(); err != nil {
		return nil, err
	}

	if err := r.loadDeleted(); err != nil {
		return nil, err
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

func (r *FileRepository) AddUserURL(userID, shortID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.mem.AddUserURL(userID, shortID); err != nil {
		return err
	}

	return r.saveUsersLocked()
}

func (r *FileRepository) ListUserURLs(userID string) ([]model.URLMapping, error) {
	return r.mem.ListUserURLs(userID)
}

func (r *FileRepository) MarkDeleted(userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.mem.MarkDeleted(userID, ids); err != nil {
		return err
	}

	return r.saveDeletedLocked()
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
	dir := filepath.Dir(r.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	items := r.mem.Items()
	records := make([]fileRecord, 0, len(items))

	i := 0
	for id, original := range items {
		i++
		records = append(records, fileRecord{
			UUID:        strconv.Itoa(i),
			ShortURL:    id,
			OriginalURL: original,
		})
	}

	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, r.path)
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

	var users map[string][]string
	if err := json.Unmarshal(b, &users); err != nil {
		return err
	}

	for userID, ids := range users {
		for _, id := range ids {
			_ = r.mem.AddUserURL(userID, id)
		}
	}

	return nil
}

func (r *FileRepository) saveUsersLocked() error {
	dir := filepath.Dir(r.usersPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	b, err := json.MarshalIndent(r.mem.UserItems(), "", "  ")
	if err != nil {
		return err
	}

	tmp := r.usersPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, r.usersPath)
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

	var deleted map[string]bool
	if err := json.Unmarshal(b, &deleted); err != nil {
		return err
	}

	for id, ok := range deleted {
		if ok {
			r.mem.setDeleted(id, true)
		}
	}

	return nil
}

func (r *FileRepository) saveDeletedLocked() error {
	dir := filepath.Dir(r.deletedPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	b, err := json.MarshalIndent(r.mem.DeletedItems(), "", "  ")
	if err != nil {
		return err
	}

	tmp := r.deletedPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, r.deletedPath)
}

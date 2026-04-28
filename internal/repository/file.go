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
	mu        sync.Mutex
	path      string
	usersPath string
	mem       *MemoryRepository
	user      map[string]map[string]struct{}
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
		path:      path,
		usersPath: path + ".users",
		mem:       NewMemoryRepository(),
		user:      make(map[string]map[string]struct{}),
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	if err := r.loadUsers(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *FileRepository) Get(id string) (string, bool) {
	return r.mem.Get(id)
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

	set, ok := r.user[userID]
	if !ok {
		set = make(map[string]struct{})
		r.user[userID] = set
	}

	set[shortID] = struct{}{}
	return r.saveUsersLocked()
}

func (r *FileRepository) ListUserURLs(userID string) ([]model.URLMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil, nil
	}

	out := make([]model.URLMapping, 0, len(set))
	for id := range set {
		orig, ok := r.mem.Get(id)
		if !ok {
			continue
		}

		out = append(out, model.URLMapping{
			ID:       id,
			Original: orig,
		})
	}

	return out, nil
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

	var m map[string][]string
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	for userID, ids := range m {
		set := make(map[string]struct{})
		for _, id := range ids {
			if id == "" {
				continue
			}

			set[id] = struct{}{}
		}

		r.user[userID] = set
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

	m := make(map[string][]string, len(r.user))
	for userID, set := range r.user {
		ids := make([]string, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		m[userID] = ids
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	tmp := r.usersPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, r.usersPath)
}

package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// FileRepository stores URL mappings on disk as JSON.
type FileRepository struct {
	mu   sync.RWMutex
	path string
	data map[string]string
	seq  int

	usersPath string
	user      map[string]map[string]struct{}
}

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// NewFileRepository creates a file-backed repository and loads data if the file exists.
func NewFileRepository(path string) (*FileRepository, error) {
	if path == "" {
		return nil, errors.New("empty file storage path")
	}

	r := &FileRepository{
		path:      path,
		data:      make(map[string]string),
		usersPath: path + ".users",
		user:      make(map[string]map[string]struct{}),
	}

	if err := r.load(); err != nil {
		return nil, err
	}
	if err := r.loadUsers(); err != nil {
		return nil, err
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *FileRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[id]

	return v, ok
}

func (r *FileRepository) Put(id string, original string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[id] = original
	_ = r.saveLocked()
}

func (r *FileRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.data[id]

	return ok
}

func (r *FileRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

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

	maxUUID := 0
	for _, rec := range recs {
		r.data[rec.ShortURL] = rec.OriginalURL

		if n, ok := atoi(rec.UUID); ok && n > maxUUID {
			maxUUID = n
		}
	}
	r.seq = maxUUID

	return nil
}

func (r *FileRepository) saveLocked() error {
	dir := filepath.Dir(r.path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	records := make([]fileRecord, 0, len(r.data))
	i := 0
	for id, original := range r.data {
		i++
		records = append(records, fileRecord{
			UUID:        itoa(i),
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

func atoi(s string) (int, bool) {
	n := 0

	if s == "" {
		return 0, false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}

	return n, true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
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

	for uid, ids := range m {
		set := make(map[string]struct{})
		for _, id := range ids {
			if id == "" {
				continue
			}
			set[id] = struct{}{}
		}
		r.user[uid] = set
	}

	return nil
}

func (r *FileRepository) saveUsersLocked() error {
	dir := filepath.Dir(r.usersPath)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	m := make(map[string][]string, len(r.user))
	for uid, set := range r.user {
		ids := make([]string, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		m[uid] = ids
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

func (r *FileRepository) ListUserURLs(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil, nil
	}

	out := make([]UserURL, 0, len(set))
	for id := range set {
		orig, ok := r.data[id]
		if !ok {
			continue
		}
		out = append(out, UserURL{ShortID: id, Original: orig})
	}

	return out, nil
}

func (r *FileRepository) CountURLs() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.data), nil
}

func (r *FileRepository) CountUsers() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.user), nil
}

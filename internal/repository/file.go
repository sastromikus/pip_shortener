package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type FileRepository struct {
	mu   sync.Mutex
	path string
	data map[string]string
	seq  int
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
		path: path,
		data: make(map[string]string),
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *FileRepository) Get(id string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.data[id]
	return v, ok
}

func (r *FileRepository) PutIfAbsent(id string, original string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; ok {
		return false
	}

	r.data[id] = original
	_ = r.saveLocked()
	return true
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

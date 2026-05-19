package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type FileRepository struct {
	mu      sync.Mutex
	path    string
	data    map[string]string
	uuidSeq int
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

func (r *FileRepository) GetByOriginal(original string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, storedOriginal := range r.data {
		if storedOriginal == original {
			return id, true
		}
	}

	return "", false
}

func (r *FileRepository) PutIfAbsent(id string, original string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; ok {
		return false, nil
	}

	r.data[id] = original
	if err := r.saveLocked(); err != nil {
		delete(r.data, id)
		return false, err
	}

	return true, nil
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
		if n, err := strconv.Atoi(rec.UUID); err == nil && n > maxUUID {
			maxUUID = n
		}
	}
	r.uuidSeq = maxUUID

	return nil
}

func (r *FileRepository) saveLocked() error {
	dir := filepath.Dir(r.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	records := make([]fileRecord, 0, len(r.data))
	i := 0
	for id, original := range r.data {
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

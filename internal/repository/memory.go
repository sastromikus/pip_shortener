package repository

import (
	"sync"

	"github.com/sastromikus/pip_shortener/internal/model"
)

type MemoryRepository struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data: make(map[string]string),
	}
}

func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.data[id]
	return v, ok
}

func (r *MemoryRepository) GetByOriginal(original string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, storedOriginal := range r.data {
		if storedOriginal == original {
			return id, true
		}
	}

	return "", false
}

func (r *MemoryRepository) PutIfAbsent(id string, original string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; ok {
		return false, nil
	}

	r.data[id] = original
	return true, nil
}

func (r *MemoryRepository) PutBatchIfAbsent(items []model.URLItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		if _, ok := r.data[item.ID]; ok {
			continue
		}

		r.data[item.ID] = item.Original
	}

	return nil
}

func (r *MemoryRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, id)
}

func (r *MemoryRepository) Items() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make(map[string]string, len(r.data))
	for id, original := range r.data {
		out[id] = original
	}

	return out
}

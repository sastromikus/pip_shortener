package repository

import "sync"

type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

type MemoryRepository struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data: make(map[string]string),
	}
}

func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[id]
	return v, ok
}

func (r *MemoryRepository) Put(id string, original string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[id] = original
}

func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.data[id]
	return ok
}
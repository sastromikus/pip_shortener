package repository

import "sync"

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

func (r *MemoryRepository) PutIfAbsent(id string, original string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; ok {
		return false, nil
	}

	r.data[id] = original
	return true, nil
}

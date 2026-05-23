package repository

import (
	"sync"

	"github.com/sastromikus/pip_shortener/internal/model"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	data    map[string]string
	deleted map[string]bool
	user    map[string]map[string]struct{}
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data:    make(map[string]string),
		deleted: make(map[string]bool),
		user:    make(map[string]map[string]struct{}),
	}
}

func (r *MemoryRepository) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.data[id]
	return v, ok
}

func (r *MemoryRepository) GetWithDeleted(id string) (string, bool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	original, ok := r.data[id]
	if !ok {
		return "", false, false
	}

	return original, true, r.deleted[id]
}

func (r *MemoryRepository) GetByOriginal(original string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *MemoryRepository) AddUserURL(userID, shortID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.user == nil {
		r.user = make(map[string]map[string]struct{})
	}

	set, ok := r.user[userID]
	if !ok {
		set = make(map[string]struct{})
		r.user[userID] = set
	}

	set[shortID] = struct{}{}
	return nil
}

func (r *MemoryRepository) ListUserURLs(userID string) ([]model.URLMapping, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil, nil
	}

	out := make([]model.URLMapping, 0, len(set))
	for shortID := range set {
		orig, ok := r.data[shortID]
		if !ok {
			continue
		}

		out = append(out, model.URLMapping{
			ID:       shortID,
			Original: orig,
		})
	}

	return out, nil
}

func (r *MemoryRepository) MarkDeleted(userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil
	}

	if r.deleted == nil {
		r.deleted = make(map[string]bool)
	}

	for _, id := range ids {
		if _, ok := set[id]; ok {
			r.deleted[id] = true
		}
	}

	return nil
}

func (r *MemoryRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.data, id)
	delete(r.deleted, id)
	for _, set := range r.user {
		delete(set, id)
	}
}

func (r *MemoryRepository) Items() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]string, len(r.data))
	for id, original := range r.data {
		out[id] = original
	}

	return out
}

func (r *MemoryRepository) UserItems() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string][]string, len(r.user))
	for userID, set := range r.user {
		ids := make([]string, 0, len(set))
		for id := range set {
			ids = append(ids, id)
		}
		out[userID] = ids
	}

	return out
}

func (r *MemoryRepository) DeletedItems() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]bool, len(r.deleted))
	for id, deleted := range r.deleted {
		out[id] = deleted
	}

	return out
}

func (r *MemoryRepository) setDeleted(id string, deleted bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.deleted == nil {
		r.deleted = make(map[string]bool)
	}

	r.deleted[id] = deleted
}

package repository

import (
	"sort"
	"sync"

	"github.com/sastromikus/pip_shortener/internal/model"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	data     map[string]string
	original map[string]string
	user     map[string]map[string]struct{}
	deleted  map[string]bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data:     make(map[string]string),
		original: make(map[string]string),
		user:     make(map[string]map[string]struct{}),
		deleted:  make(map[string]bool),
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

	v, ok := r.data[id]
	if !ok {
		return "", false, false
	}

	return v, true, r.deleted[id]
}

func (r *MemoryRepository) GetByOriginal(original string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.original[original]
	return id, ok
}

func (r *MemoryRepository) PutIfAbsent(id string, original string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; ok {
		return false, nil
	}
	if _, ok := r.original[original]; ok {
		return false, nil
	}

	r.data[id] = original
	r.original[original] = id
	return true, nil
}

func (r *MemoryRepository) PutBatchIfAbsent(items []model.URLItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		if _, ok := r.data[item.ID]; ok {
			continue
		}
		if _, ok := r.original[item.Original]; ok {
			continue
		}

		r.data[item.ID] = item.Original
		r.original[item.Original] = item.ID
	}

	return nil
}

// Put is kept for tests and simple compatibility. Business code should prefer PutIfAbsent.
func (r *MemoryRepository) Put(id string, original string) {
	_, _ = r.PutIfAbsent(id, original)
}

func (r *MemoryRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.data[id]
	return ok
}

func (r *MemoryRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if original, ok := r.data[id]; ok {
		delete(r.original, original)
	}
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
		sort.Strings(ids)
		out[userID] = ids
	}

	return out
}

func (r *MemoryRepository) DeletedItems() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.deleted))
	for id, deleted := range r.deleted {
		if deleted {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	return ids
}

func (r *MemoryRepository) AddUserURL(userID, shortID string) error {
	return r.AddUserURLs(userID, []string{shortID})
}

func (r *MemoryRepository) AddUserURLs(userID string, shortIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if userID == "" || len(shortIDs) == 0 {
		return nil
	}

	set := r.user[userID]
	if set == nil {
		set = make(map[string]struct{})
		r.user[userID] = set
	}
	for _, id := range shortIDs {
		if id != "" {
			set[id] = struct{}{}
		}
	}

	return nil
}

func (r *MemoryRepository) LoadUserURL(userID, shortID string) {
	_ = r.AddUserURL(userID, shortID)
}

func (r *MemoryRepository) ListUserURLs(userID string) ([]model.UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]model.UserURL, 0, len(ids))
	for _, id := range ids {
		original, ok := r.data[id]
		if !ok || r.deleted[id] {
			continue
		}
		out = append(out, model.UserURL{ShortID: id, Original: original})
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
	for _, id := range ids {
		if _, ok := set[id]; ok {
			r.deleted[id] = true
		}
	}

	return nil
}

func (r *MemoryRepository) LoadDeleted(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id != "" {
		r.deleted[id] = true
	}
}

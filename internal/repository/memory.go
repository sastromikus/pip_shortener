package repository

import "sync"

type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

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

func (r *MemoryRepository) ListUserURLs(userID string) ([]UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	set := r.user[userID]
	if len(set) == 0 {
		return nil, nil
	}

	out := make([]UserURL, 0, len(set))
	for shortID := range set {
		orig, ok := r.data[shortID]
		if !ok {
			continue
		}
		out = append(out, UserURL{ShortID: shortID, Original: orig})
	}

	return out, nil
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

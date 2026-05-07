package repository

import "sync"

// URLRepository defines a storage backend for short URL mappings.
type URLRepository interface {
	Get(id string) (string, bool)
	Put(id string, original string)
	Exists(id string) bool
}

// MemoryRepository stores URL mappings in memory.
type MemoryRepository struct {
	mu   sync.RWMutex
	data map[string]string
	user map[string]map[string]struct{}
}

// NewMemoryRepository creates a new in-memory repository.
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

func (r *MemoryRepository) CountURLs() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.data), nil
}

func (r *MemoryRepository) CountUsers() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.user == nil {
		return 0, nil
	}
	return len(r.user), nil
}

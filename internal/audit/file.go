package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

// FileObserver appends audit events to a file as JSON lines.
type FileObserver struct {
	mu   sync.Mutex
	path string
}

// NewFileObserver creates a file-based audit observer.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify writes one audit event to the configured file.
func (o *FileObserver) Notify(ctx context.Context, e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	f, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(b, '\n'))
	return err
}

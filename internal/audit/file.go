package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

// FileObserver appends audit events to a file as JSON lines.
type FileObserver struct {
	mu sync.Mutex
	f  *os.File
}

// NewFileObserver creates a file-based audit observer and opens the target file once.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{f: f}, nil
}

// Notify writes one audit event to the configured file.
func (o *FileObserver) Notify(ctx context.Context, e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	_, err = o.f.Write(append(b, '\n'))
	return err
}

// Close closes the underlying audit file.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.f == nil {
		return nil
	}
	err := o.f.Close()
	o.f = nil
	return err
}

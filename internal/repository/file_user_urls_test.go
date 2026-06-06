package repository

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileRepository_UserURLsAndCounts(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "storage.json")

	r, err := NewFileRepository(p)
	if err != nil {
		t.Fatalf("NewFileRepository: %v", err)
	}

	r.Put("A1", "https://a")
	r.Put("B2", "https://b")

	_ = r.AddUserURL(context.Background(), "u1", "A1")
	_ = r.AddUserURL(context.Background(), "u1", "B2")

	urls, users, _ := r.Stats(context.Background())

	if urls != 2 {
		t.Fatalf("CountURLs: want 2, got %d", urls)
	}
	if users != 1 {
		t.Fatalf("CountUsers: want 1, got %d", users)
	}

	r2, err := NewFileRepository(p)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	urls2, _, _ := r2.Stats(context.Background())
	if urls2 != 2 {
		t.Fatalf("reopen CountURLs: want 2, got %d", urls2)
	}
}

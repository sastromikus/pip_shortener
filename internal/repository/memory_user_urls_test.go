package repository

import (
	"context"
	"testing"
)

func TestMemoryRepository_UserURLsAndCounts(t *testing.T) {
	r := NewMemoryRepository()

	r.Put("A1", "https://a")
	r.Put("B2", "https://b")

	_ = r.AddUserURL(context.Background(), "u1", "A1")
	_ = r.AddUserURL(context.Background(), "u1", "B2")
	_ = r.AddUserURL(context.Background(), "u2", "B2")

	urls, users, err := r.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if urls != 2 {
		t.Fatalf("CountURLs: want 2, got %d", urls)
	}

	if users != 2 {
		t.Fatalf("CountUsers: want 2, got %d", users)
	}

	list, err := r.ListUserURLs(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListUserURLs: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListUserURLs: want 2, got %d", len(list))
	}
}

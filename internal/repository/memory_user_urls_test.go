package repository

import "testing"

func TestMemoryRepository_UserURLsAndCounts(t *testing.T) {
	r := NewMemoryRepository()

	r.Put("A1", "https://a")
	r.Put("B2", "https://b")

	_ = r.AddUserURL("u1", "A1")
	_ = r.AddUserURL("u1", "B2")
	_ = r.AddUserURL("u2", "B2")

	urls, err := r.CountURLs()
	if err != nil {
		t.Fatalf("CountURLs: %v", err)
	}
	if urls != 2 {
		t.Fatalf("CountURLs: want 2, got %d", urls)
	}

	users, err := r.CountUsers()
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if users != 2 {
		t.Fatalf("CountUsers: want 2, got %d", users)
	}

	list, err := r.ListUserURLs("u1")
	if err != nil {
		t.Fatalf("ListUserURLs: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListUserURLs: want 2, got %d", len(list))
	}
}
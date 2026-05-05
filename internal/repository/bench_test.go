package repository

import "testing"

func BenchmarkMemoryRepository_PutGet(b *testing.B) {
	r := NewMemoryRepository()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := "ID" + itoa(i%100000)
		r.Put(id, "http://example.com")
		_, _ = r.Get(id)
	}
}

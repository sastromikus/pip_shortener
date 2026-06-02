package repository

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkMemoryRepositoryPutGet(b *testing.B) {
	r := NewMemoryRepository()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	i := 0
	for b.Loop() {
		id := "ID" + strconv.Itoa(i%100000)
		r.Put(id, "http://example.com/"+strconv.Itoa(i))
		_, _ = r.Get(ctx, id)
		i++
	}
}

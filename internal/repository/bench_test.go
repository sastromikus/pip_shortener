package repository

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkMemoryRepositoryPutGet(b *testing.B) {
	b.StopTimer()
	r := NewMemoryRepository()
	ctx := context.Background()
	i := 0
	b.ReportAllocs()
	b.ResetTimer()
	b.StartTimer()

	for b.Loop() {
		id := "ID" + strconv.Itoa(i%100000)
		r.Put(id, "http://example.com/"+strconv.Itoa(i))
		_, _ = r.Get(ctx, id)
		i++
	}
}

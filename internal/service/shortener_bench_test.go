package service

import (
	"fmt"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
)

func BenchmarkShorten(b *testing.B) {
	b.StopTimer()
	repo := repository.NewMemoryRepository()
	svc := NewShortener(repo)
	i := 0
	b.ReportAllocs()
	b.ResetTimer()
	b.StartTimer()

	for b.Loop() {
		_, _ = svc.Shorten(fmt.Sprintf("http://example.com/benchmark/%d", i))
		i++
	}
}

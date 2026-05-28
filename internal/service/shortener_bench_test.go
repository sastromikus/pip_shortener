package service

import (
	"fmt"
	"testing"

	"github.com/sastromikus/pip_shortener/internal/repository"
)

func BenchmarkShorten(b *testing.B) {
	repo := repository.NewMemoryRepository()
	svc := NewShortener(repo)

	b.ReportAllocs()
	b.ResetTimer()

	i := 0
	for b.Loop() {
		_, _ = svc.Shorten(fmt.Sprintf("http://example.com/benchmark/%d", i))
		i++
	}
}

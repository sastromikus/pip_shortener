package pool

import "sync"

// Resettable is implemented by objects that can reset their state.
type Resettable interface {
	Reset()
}

// Pool stores reusable resettable objects.
type Pool[T Resettable] struct {
	p sync.Pool
}

// New creates a new Pool.
func New[T Resettable](newFn func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get returns an object from the pool.
func (pl *Pool[T]) Get() T {
	return pl.p.Get().(T)
}

// Put resets obj and returns it to the pool.
func (pl *Pool[T]) Put(obj T) {
	obj.Reset()
	pl.p.Put(obj)
}

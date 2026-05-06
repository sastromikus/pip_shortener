package pool

import "sync"

type Resettable interface {
	Reset()
}

type Pool[T Resettable] struct {
	p sync.Pool
}

func New[T Resettable](newFn func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

func (pl *Pool[T]) Get() T {
	v := pl.p.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

func (pl *Pool[T]) Put(obj T) {
	var zero T
	if any(obj) == any(zero) {
		return
	}

	obj.Reset()
	pl.p.Put(obj)
}

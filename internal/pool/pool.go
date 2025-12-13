package pool

import "sync"

// Resettable определяет интерфейс для типов с методом Reset
type Resettable interface {
	Reset()
}

// Pool представляет контейнер для переиспользования объектов с методом Reset
type Pool[T Resettable] struct {
	pool sync.Pool
}

// New создает новый Pool для объектов типа T
func New[T Resettable](fn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return fn()
			},
		},
	}
}

// Get возвращает объект из пула
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put помещает объект в пул после вызова Reset
func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}

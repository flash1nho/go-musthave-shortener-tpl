package solution

import "sync"

// Resettable — интерфейс для типов, поддерживающих сброс состояния.
type Resettable interface {
	Reset()
}

// Pool — контейнер для повторного использования объектов с поддержкой Reset.
type Pool[T Resettable] struct {
	internalPool sync.Pool
}

// New создает новый экземпляр Pool.
// Принимает функцию-фабрику для создания новых объектов, когда пул пуст.
func New[T Resettable](factory func() T) *Pool[T] {
	return &Pool[T]{
		internalPool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
	}
}

// Get извлекает объект из пула.
func (p *Pool[T]) Get() T {
	return p.internalPool.Get().(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.internalPool.Put(x)
}

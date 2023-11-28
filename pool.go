package pr3

import (
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	elems    []PoolElem[T]
	capacity uint32
	fallback sync.Pool
}

type PoolElem[T any] struct {
	from  *sync.Pool
	refs  atomic.Int32
	value T
}

func (p *PoolElem[T]) Put() bool {
	if c := p.refs.Add(-1); c == 0 {
		if p.from != nil {
			p.from.Put(p)
		}
		return true
	} else if c < 0 {
		panic("bad put")
	}
	return false
}

func (p *PoolElem[T]) Inc() {
	p.refs.Add(1)
}

func NewPool[T any](
	capacity uint32,
	newFunc func() T,
) *Pool[T] {
	pool := &Pool[T]{
		capacity: capacity,
	}

	pool.fallback = sync.Pool{
		New: func() any {
			elem := &PoolElem[T]{
				value: newFunc(),
				from:  &pool.fallback,
			}
			return elem
		},
	}

	elems := make([]PoolElem[T], capacity)
	for i := uint32(0); i < capacity; i++ {
		i := i
		ptr := newFunc()
		elems[i] = PoolElem[T]{
			value: ptr,
		}
	}
	pool.elems = elems

	return pool
}

func (p *Pool[T]) Get(ptr *T) *PoolElem[T] {

	idx := fastrand() % p.capacity
	if p.elems[idx].refs.CompareAndSwap(0, 1) {
		*ptr = p.elems[idx].value
		return &p.elems[idx]
	}

	// fallback
	elem := p.fallback.Get().(*PoolElem[T])
	elem.refs.Store(1)
	*ptr = elem.value
	return elem
}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32

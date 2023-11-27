package pr3

import (
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	l        sync.Mutex
	newFunc  func() T
	elems    []_PoolElem[T]
	capacity uint32
	fallback sync.Pool
}

type _PoolElem[T any] struct {
	refs   atomic.Int32
	put    func() bool
	incRef func()
	value  T
}

func NewPool[T any](
	capacity uint32,
	newFunc func() T,
) *Pool[T] {
	pool := &Pool[T]{
		capacity: capacity,
		newFunc:  newFunc,
		fallback: sync.Pool{
			New: func() any {
				var elem _PoolElem[T]
				elem.value = newFunc()
				elem.put = func() bool {
					if c := elem.refs.Add(-1); c == 0 {
						return true
					} else if c < 0 {
						panic("bad put")
					}
					return false
				}
				elem.incRef = func() {
					elem.refs.Add(1)
				}
				return &elem
			},
		},
	}

	elems := make([]_PoolElem[T], capacity)
	for i := uint32(0); i < capacity; i++ {
		i := i
		ptr := newFunc()
		elems[i] = _PoolElem[T]{
			value: ptr,
			put: func() bool {
				if c := elems[i].refs.Add(-1); c == 0 {
					return true
				} else if c < 0 {
					panic("bad put")
				}
				return false
			},
			incRef: func() {
				elems[i].refs.Add(1)
			},
		}
	}
	pool.elems = elems

	return pool
}

func (p *Pool[T]) Get(ptr *T) (put func() bool) {
	put, _ = p.GetRC(ptr)
	return
}

func (p *Pool[T]) GetRC(ptr *T) (
	put func() bool,
	incRef func(),
) {

	for i := 0; i < 16; i++ {
		idx := fastrand() % p.capacity
		if p.elems[idx].refs.CompareAndSwap(0, 1) {
			*ptr = p.elems[idx].value
			put = p.elems[idx].put
			incRef = p.elems[idx].incRef
			return
		}
	}

	// fallback
	elem := p.fallback.Get().(*_PoolElem[T])
	elem.refs.Store(1)
	*ptr = elem.value
	put = elem.put
	incRef = elem.incRef
	return
}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32

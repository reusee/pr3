package pr3

import (
	"sync"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	l        sync.Mutex
	newFunc  func() T
	elems    atomic.Pointer[[]_PoolElem[T]]
	capacity uint32
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
	}
	pool.allocElems(nil)
	return pool
}

func (p *Pool[T]) allocElems(old *[]_PoolElem[T]) {
	p.l.Lock()
	defer p.l.Unlock()
	if old != nil && p.elems.Load() != old {
		// refreshed
		return
	}
	elems := make([]_PoolElem[T], p.capacity)
	for i := uint32(0); i < p.capacity; i++ {
		i := i
		ptr := p.newFunc()
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
	p.elems.Store(&elems)
}

func (p *Pool[T]) Get(ptr *T) (put func() bool) {
	put, _ = p.GetRC(ptr)
	return
}

func (p *Pool[T]) GetRC(ptr *T) (
	put func() bool,
	incRef func(),
) {

	for {
		cur := p.elems.Load()
		elems := *cur
		for i := 0; i < 16; i++ {
			idx := fastrand() % p.capacity
			if elems[idx].refs.CompareAndSwap(0, 1) {
				*ptr = elems[idx].value
				put = elems[idx].put
				incRef = elems[idx].incRef
				return
			}
		}

		p.allocElems(cur)
	}

}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32

package pr3

import (
	"runtime"
	"sync/atomic"
	_ "unsafe"
)

type Pool[T any] struct {
	shards      []*PoolElem[T]
	maxPerShard int
	newFunc     func() T
}

type PoolElem[T any] struct {
	refs  atomic.Int32
	value T
	next  *PoolElem[T]
	num   int
}

func (p *Pool[T]) Put(elem *PoolElem[T]) bool {
	if c := elem.refs.Add(-1); c == 0 {
		procID := procPin()
		if procID < len(p.shards) {
			if head := p.shards[procID]; head == nil {
				elem.num = 0
				elem.next = nil
				p.shards[procID] = elem
			} else if head.num < p.maxPerShard {
				elem.num = head.num + 1
				elem.next = head
				p.shards[procID] = elem
			}
		}
		procUnpin()
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
	capacity int,
	newFunc func() T,
) *Pool[T] {
	maxPerShard := capacity / runtime.NumCPU()

	pool := &Pool[T]{
		maxPerShard: maxPerShard,
		newFunc:     newFunc,
	}

	pool.shards = make([]*PoolElem[T], runtime.NumCPU())

	return pool
}

func (p *Pool[T]) Get(ptr *T) (ret *PoolElem[T]) {
	procID := procPin()
	if procID < len(p.shards) && p.shards[procID] != nil {
		ret = p.shards[procID]
		p.shards[procID] = p.shards[procID].next
	}
	procUnpin()
	if ret == nil {
		ret = &PoolElem[T]{
			value: p.newFunc(),
		}
	}
	ret.refs.Store(1)
	*ptr = ret.value
	return
}

//go:linkname procPin runtime.procPin
func procPin() int

//go:linkname procUnpin runtime.procUnpin
func procUnpin()

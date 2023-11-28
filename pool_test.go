package pr3

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestBytesPool(t *testing.T) {
	pool := NewPool(8, func() []byte {
		return make([]byte, 8)
	})
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var bs []byte
				elem := pool.Get(&bs)
				defer pool.Put(elem)
				binary.PutUvarint(bs, uint64(i))
			}
		}()
	}
	wg.Wait()
}

func TestBytesPoolRC(t *testing.T) {
	pool := NewPool(8, func() []byte {
		return make([]byte, 8)
	})
	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var bs []byte
				elem := pool.Get(&bs)
				defer pool.Put(elem)
				nRef := rand.Intn(16)
				for i := 0; i < nRef; i++ {
					elem.Inc()
				}
				defer func() {
					for i := 0; i < nRef; i++ {
						elem.Inc()
					}
				}()
				binary.PutUvarint(bs, uint64(i))
			}
		}()
	}
	wg.Wait()
}

func TestBytesPoolRCOverload(t *testing.T) {
	pool := NewPool(1, func() int {
		return 42
	})
	var i int
	pool.Get(&i)
	var j int
	elem := pool.Get(&j)
	elem.Inc()
	if pool.Put(elem) {
		t.Fatal()
	}
	if !pool.Put(elem) {
		t.Fatal()
	}
}

func BenchmarkBytesPool(b *testing.B) {
	pool := NewPool(8, func() []byte {
		return make([]byte, 8)
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []byte
		elem := pool.Get(&v)
		pool.Put(elem)
	}
}

func BenchmarkParallelBytesPool(b *testing.B) {
	pool := NewPool(1024, func() []byte {
		return make([]byte, 8)
	})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v []byte
			elem := pool.Get(&v)
			pool.Put(elem)
		}
	})
}

func TestPoolBadPut(t *testing.T) {
	pool := NewPool(1, func() int {
		return 42
	})
	var i int
	elem := pool.Get(&i)
	pool.Put(elem)
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal()
			}
			if fmt.Sprintf("%v", p) != "bad put" {
				t.Fatal()
			}
		}()
		pool.Put(elem)
	}()
}

func TestPoolBadPutRC(t *testing.T) {
	pool := NewPool(1, func() int {
		return 42
	})
	var j int
	pool.Get(&j)
	var i int
	elem := pool.Get(&i)
	pool.Put(elem)
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal()
			}
			if fmt.Sprintf("%v", p) != "bad put" {
				t.Fatal()
			}
		}()
		pool.Put(elem)
	}()
}

func BenchmarkPoolDrain(b *testing.B) {
	pool := NewPool(1, func() []byte {
		return make([]byte, 8)
	})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v []byte
			elem := pool.Get(&v)
			pool.Put(elem)
		}
	})
}

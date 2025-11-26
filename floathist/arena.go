package floathist

import (
	"math/bits"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	lBatch = 1024
	lAlloc = 8

	ptrSize = unsafe.Sizeof(unsafe.Pointer(nil))
)

type arena[V any] struct {
	_ [0]func() // no equality

	s atomic.Pointer[*[lBatch]V]
	p atomic.Uint32
	t atomic.Uint32

	mu sync.Mutex // protects realloc
}

func (t *arena[V]) Allocated() uint32 { return t.p.Load() }

type tag[V any] struct{}

var x atomic.Pointer[int]

type arenaP[V any] struct {
	_ tag[V]
	v uint32
}

func arenaRaw[V any](v uint32) arenaP[V] { return arenaP[V]{v: v} }
func (p arenaP[V]) Raw() uint32          { return p.v }

func (l *arena[V]) Get(p arenaP[V]) *V {
	b := unsafe.Add(unsafe.Pointer(l.s.Load()), uintptr(p.v/lBatch)*ptrSize)
	return &(*(**[lBatch]V)(b))[p.v%lBatch]
}

func (l *arena[V]) New() (p arenaP[V]) {
	p.v = l.p.Add(1)
	if p.v >= l.t.Load() {
		l.realloc(p.v)
	}
	return
}

func (l *arena[V]) realloc(v uint32) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for t := l.t.Load(); v >= t; t += lBatch {
		var arr []*[lBatch]V

		switch {
		// first time through we initally alloc lAlloc
		case t == 0:
			arr = make([]*[lBatch]V, lAlloc)
			l.s.Store(&arr[0])

		// we need to reallocate if we're at at least lBatch*lAlloc and we've
		// just hit a new power of 2
		case t >= lBatch*lAlloc && bits.OnesCount32(t) == 1:
			arr = make([]*[lBatch]V, t/(lBatch/2))
			copy(arr, unsafe.Slice(l.s.Load(), t/lBatch))
			l.s.Store(&arr[0])

		// otherwise, load arr so we can allocate a new batch
		default:
			arr = unsafe.Slice(l.s.Load(), t/lBatch+1)
		}

		arr[t/lBatch] = new([lBatch]V)
		l.t.Add(lBatch) // synchronizes arr reads
	}
}

package hydrant

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	threadHint     atomic.Uint64
	threadHintPool = sync.Pool{New: func() any { return threadHint.Add(1) - 1 }}

	spanList = make([]listRoot, runtime.NumCPU())
)

type listRoot struct {
	listRootData
	_ [128 - unsafe.Sizeof(listRootData{})]byte
}

type listRootData struct {
	head atomic.Pointer[Span]
	mu   sync.Mutex
}

func IterateSpans(f func(*Span) bool) {
	roots := make([]*Span, len(spanList))
	for i := range spanList {
		roots[i] = spanList[i].head.Load()
	}
	for _, span := range roots {
		for ; span != nil; span = span.next.Load() {
			if !span.done.Load() && !f(span) {
				return
			}
		}
	}
}

func pushSpan(s *Span) *listRoot {
	roots := spanList
	if len(roots) == 0 {
		return nil
	}

	hintAny := threadHintPool.Get()
	hint, _ := hintAny.(uint64)
	root := &roots[hint%uint64(len(roots))]

	root.mu.Lock()
	if head := root.head.Load(); head != nil {
		s.next.Store(head)
		head.prev.Store(s)
	}
	root.head.Store(s)
	root.mu.Unlock()

	threadHintPool.Put(hintAny)

	return root
}

func popSpan(s *Span) {
	if s == nil || s.root == nil {
		return
	}

	s.root.mu.Lock()
	prev, next := s.prev.Load(), s.next.Load()
	if prev != nil {
		prev.next.Store(next)
	} else {
		s.root.head.Store(next)
	}
	if next != nil {
		next.prev.Store(prev)
	}
	s.root.mu.Unlock()
}

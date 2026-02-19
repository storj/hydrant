package utils

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
)

type ringSlot[T any] struct {
	ptr atomic.Pointer[T]
	seq atomic.Uint64
}

type RingBuffer[T any] struct {
	slots []ringSlot[T]
	pos   atomic.Uint64

	wmu      sync.Mutex // held only by Watch to update the watcher list
	watchers atomic.Pointer[[]chan<- T]
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	rb := &RingBuffer[T]{slots: make([]ringSlot[T], size)}
	rb.watchers.Store(new([]chan<- T))
	return rb
}

// Add adds an item to the buffer, overwriting the oldest item if the buffer is full.
func (r *RingBuffer[T]) Add(v T) {
	n := uint64(len(r.slots))
	i := r.pos.Add(1) - 1
	s := &r.slots[i%n]
	s.ptr.Store(&v)
	s.seq.Store(i + 1) // mark slot as written for position i

	if ws := r.watchers.Load(); len(*ws) != 0 {
		for _, ch := range *ws {
			select {
			case ch <- v:
			default: // watcher is slow, drop
			}
		}
	}
}

// Get returns a copy of the current contents of the buffer, ordered from oldest to newest.
func (r *RingBuffer[T]) Get() []T {
	pos := r.pos.Load()
	n := uint64(len(r.slots))
	count := min(pos, n)
	out := make([]T, 0, count)
	for i := pos - count; i < pos; i++ {
		s := &r.slots[i%n]
		if s.seq.Load() == i+1 {
			out = append(out, *s.ptr.Load())
		}
	}
	return out
}

// Watch calls cb with each item Added to the buffer and blocks until the context is done.
func (r *RingBuffer[T]) Watch(ctx context.Context, cb func(T)) {
	ch := make(chan T, 64)

	r.wmu.Lock()
	next := append(*r.watchers.Load(), ch)
	r.watchers.Store(&next)
	r.wmu.Unlock()

	defer func() {
		r.wmu.Lock()
		defer r.wmu.Unlock()

		next := slices.DeleteFunc(
			slices.Clone(*r.watchers.Load()),
			func(w chan<- T) bool { return w == ch },
		)
		r.watchers.Store(&next)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case v := <-ch:
			cb(v)
		}
	}
}

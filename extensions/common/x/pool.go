package x

import (
	"github.com/puzpuzpuz/xsync/v2"
)

const DefaultPoolCapacity = 2048

// x.Pool is a specialized object pool implementation to replace sync.Pool.
// Unlike sync.Pool, x.Pool is backed by an MPMC queue, and objects in x.Pool
// will not be garbage collected for the lifetime of the pool, and its capacity is bounded.
// This is more useful if the objects in the pool should be long-lived.
type Pool[T comparable] struct {
	zero    T
	factory func() T
	q       *xsync.MPMCQueue
}

func NewPool[T comparable](capacity int) *Pool[T] {
	p := &Pool[T]{
		q: xsync.NewMPMCQueue(capacity),
	}
	return p
}

func (x *Pool[T]) WithFactory(factory func() T) *Pool[T] {
	x.factory = factory
	return x
}

func (x *Pool[T]) Get() T {
	item, ok := x.q.TryDequeue()
	if ok {
		return item.(T)
	}
	return x.factory()
}

func (x *Pool[T]) Put(item T) {
	if item == x.zero {
		panic("x.Pool: cannot put zero value into the pool")
	}
	x.q.TryEnqueue(item)
}

package common

import (
	"context"
	"io"
	"sync"

	"golang.org/x/sync/semaphore"
)

type IOContext struct {
	ctx                  context.Context
	waitgroup            sync.WaitGroup
	hdrPool              *HeadersProxyPool
	limiter              *semaphore.Weighted
	nativeReaderWrappers []*NativeReaderWrapper
	cleanupFuncs         []func()
}

func newIOContext(concurrent int64) *IOContext {
	return &IOContext{
		waitgroup:            sync.WaitGroup{},
		limiter:              semaphore.NewWeighted(concurrent),
		nativeReaderWrappers: make([]*NativeReaderWrapper, 0),
		cleanupFuncs:         make([]func(), 0),
	}
}

func (t *IOContext) Context() context.Context {
	return t.ctx
}

func (t *IOContext) GetHeadersProxy() *HeadersProxy {
	h := t.hdrPool.Get()
	t.RegisterCleanup(func() {
		t.hdrPool.Put(h)
	})
	return h
}

func (t *IOContext) AcquireFetchToken() (err error) {
	err = t.limiter.Acquire(t.Context(), 1)
	if err != nil {
		return
	}
	t.waitgroup.Add(1)
	return
}

func (t *IOContext) ReleaseFetchToken() {
	t.waitgroup.Done()
	t.limiter.Release(1)
}

func (t *IOContext) RegisterCleanup(c func()) {
	if c == nil {
		return
	}
	t.cleanupFuncs = append(t.cleanupFuncs, c)
}

func (t *IOContext) TrackReader(w *NativeReaderWrapper) {
	if w == nil {
		return
	}
	t.nativeReaderWrappers = append(t.nativeReaderWrappers, w)
}

func (t *IOContext) release() {
	t.waitgroup.Wait()

	buf := GetBuffer()
	defer PutBuffer(buf)

	for i := range t.nativeReaderWrappers {
		reader := t.nativeReaderWrappers[i].reader
		io.CopyBuffer(io.Discard, reader, buf)
		reader.Close()
		t.nativeReaderWrappers[i].reader = nil
	}
	t.nativeReaderWrappers = t.nativeReaderWrappers[:0]

	for i := len(t.cleanupFuncs) - 1; i >= 0; i-- {
		t.cleanupFuncs[i]()
		t.cleanupFuncs[i] = nil
	}
	t.cleanupFuncs = t.cleanupFuncs[:0]
}

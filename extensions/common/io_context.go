package common

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"go.miragespace.co/heresy/extensions/common/shared"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type IOContext struct {
	extenderMu        sync.Mutex
	extenders         int
	waiting           bool
	extensionDone     chan struct{}
	extendedCtx       context.Context
	extendedCtxCancel context.CancelFunc
	shouldExtend      atomic.Bool
	reqCtx            context.Context
	runtimeContext    context.Context
	runtimeDone       <-chan struct{}
	stopRequest       func() bool
	stopRuntime       func() bool
	ioGroup           sync.WaitGroup
	logger            *zap.Logger
	hdrPool           *shared.HeadersProxyPool
	limiter           *semaphore.Weighted
	cleanupFuncs      []func()
}

func newIOContext(logger *zap.Logger, concurrent int64) *IOContext {
	return &IOContext{
		logger:  logger.With(zap.String("component", "ioContext")),
		limiter: semaphore.NewWeighted(concurrent),
	}
}

func (t *IOContext) ExtendContext() bool {
	t.extenderMu.Lock()
	defer t.extenderMu.Unlock()
	if t.extendedCtx.Err() != nil || (t.waiting && t.extenders == 0) {
		return false
	}
	t.extenders++
	t.shouldExtend.Store(true)
	return true
}

func (t *IOContext) ConcludeExtend() {
	t.extenderMu.Lock()
	defer t.extenderMu.Unlock()
	t.extenders--
	if t.waiting && t.extenders == 0 {
		close(t.extensionDone)
	}
}

// Context covers native I/O, including work started before waitUntil extends it.
func (t *IOContext) Context() context.Context        { return t.extendedCtx }
func (t *IOContext) RequestContext() context.Context { return t.reqCtx }
func (t *IOContext) RuntimeDone() <-chan struct{}    { return t.runtimeDone }
func (t *IOContext) RuntimeCanceled() bool           { return t.runtimeContext.Err() != nil }

func (t *IOContext) GetHeadersProxy() *shared.HeadersProxy {
	h := t.hdrPool.Get()
	t.RegisterCleanup(func() { t.hdrPool.Put(h) })
	return h
}

// Register native work on the VM thread, before launching its goroutine. Keep it
// registered until its result has been delivered on the loop (or rejected by a
// terminated loop), so cleanup cannot race queued callbacks or waiting I/O.
func (t *IOContext) StartIO() { t.ioGroup.Add(1) }
func (t *IOContext) EndIO()   { t.ioGroup.Done() }
func (t *IOContext) CompleteIO(loop *eventloop.EventLoop, fn func(*goja.Runtime), discard func()) {
	if loop.RunOnLoop(func(vm *goja.Runtime) {
		defer t.EndIO()
		fn(vm)
	}) {
		return
	}
	if discard != nil {
		discard()
	}
	t.EndIO()
}
func (t *IOContext) AcquireFetchToken() error { return t.limiter.Acquire(t.Context(), 1) }
func (t *IOContext) ReleaseFetchToken()       { t.limiter.Release(1) }

func (t *IOContext) RegisterCleanup(c func()) {
	if c != nil {
		t.cleanupFuncs = append(t.cleanupFuncs, c)
	}
}

func (t *IOContext) wait() {
	t.extenderMu.Lock()
	t.waiting = true
	if t.extenders == 0 {
		close(t.extensionDone)
	}
	t.extenderMu.Unlock()
	select {
	case <-t.extensionDone:
	case <-t.runtimeDone:
	}
	t.extendedCtxCancel()
}

func (t *IOContext) release() {
	t.ioGroup.Wait()
	t.stopRequest()
	t.stopRuntime()
	for i := len(t.cleanupFuncs) - 1; i >= 0; i-- {
		t.cleanupFuncs[i]()
		t.cleanupFuncs[i] = nil
	}
	t.cleanupFuncs = t.cleanupFuncs[:0]
}

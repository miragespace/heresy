package common

import (
	"context"
	"sync"
	"sync/atomic"

	"go.miragespace.co/heresy/extensions/common/shared"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type IOContext struct {
	extenderGroup        sync.WaitGroup
	extendedCtx          context.Context
	extendedCtxCancel    context.CancelFunc
	fetchGroup           sync.WaitGroup
	shouldExtend         atomic.Bool
	reqCtx               context.Context
	logger               *zap.Logger
	hdrPool              *shared.HeadersProxyPool
	limiter              *semaphore.Weighted
	nativeReaderWrappers []*shared.NativeReaderWrapper
	cleanupFuncs         []func()
}

func newIOContext(logger *zap.Logger, concurrent int64) *IOContext {
	return &IOContext{
		logger:               logger.With(zap.String("component", "ioContext")),
		limiter:              semaphore.NewWeighted(concurrent),
		nativeReaderWrappers: make([]*shared.NativeReaderWrapper, 0),
		cleanupFuncs:         make([]func(), 0),
	}
}

func (t *IOContext) ExtendContext() {
	// t.logger.Debug("extending context")
	t.shouldExtend.Store(true)
	t.extenderGroup.Add(1)
	// From godoc: "Note that calls with a positive delta that occur
	// when the counter is zero must happen before a Wait.
	// Calls with a negative delta, or calls with a positive delta that start
	// when the counter is greater than zero, may happen at any time."
	//
	// Meaning that when sync.WaitGroup's counter is > 0, reentrant .Add(1)
	// from .ExtendContext() via .waitUntil() will not violate this invariant.
	// That means calling .waitUntil after the handler returns is a data race.
}

func (t *IOContext) ConcludeExtend() {
	// t.logger.Debug("concluding extension")
	t.extenderGroup.Done()
}

func (t *IOContext) Context() context.Context {
	if t.shouldExtend.Load() {
		// t.logger.Debug("returning extended context")
		return t.extendedCtx
	} else {
		// t.logger.Debug("returning http request context")
		return t.reqCtx
	}
}

func (t *IOContext) GetHeadersProxy() *shared.HeadersProxy {
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
	t.fetchGroup.Add(1)
	return
}

func (t *IOContext) ReleaseFetchToken() {
	t.fetchGroup.Done()
	t.limiter.Release(1)
}

func (t *IOContext) RegisterCleanup(c func()) {
	if c == nil {
		return
	}
	t.cleanupFuncs = append(t.cleanupFuncs, c)
	// caller := zapcore.NewEntryCaller(runtime.Caller(1))
	// t.logger.Debug("cleanup registed", zap.String("via", caller.TrimmedPath()))
}

func (t *IOContext) TrackReader(w *shared.NativeReaderWrapper) {
	if w == nil {
		return
	}
	t.nativeReaderWrappers = append(t.nativeReaderWrappers, w)
}

func (t *IOContext) release() {
	go func() {
		if t.shouldExtend.Load() {
			// t.logger.Debug("waiting for extenders")
			t.extenderGroup.Wait()
			t.extendedCtxCancel()
		} else {
			<-t.reqCtx.Done()
			// t.logger.Debug("http request cancelled")
			t.extendedCtxCancel()
		}
	}()

	<-t.extendedCtx.Done()
	t.fetchGroup.Wait()

	// t.logger.Debug("releasing readers", zap.Int("readers", len(t.nativeReaderWrappers)))

	buf := GetBuffer()
	defer PutBuffer(buf)

	for i := range t.nativeReaderWrappers {
		t.nativeReaderWrappers[i].Reset(buf)
	}
	t.nativeReaderWrappers = t.nativeReaderWrappers[:0]

	// t.logger.Debug("invoking cleanup", zap.Int("funcs", len(t.cleanupFuncs)))

	for i := len(t.cleanupFuncs) - 1; i >= 0; i-- {
		t.cleanupFuncs[i]()
		t.cleanupFuncs[i] = nil
	}
	t.cleanupFuncs = t.cleanupFuncs[:0]
}

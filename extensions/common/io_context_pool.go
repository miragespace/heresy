package common

import (
	"context"
	"expvar"

	"go.miragespace.co/heresy/extensions/common/shared"
	"go.miragespace.co/heresy/extensions/common/x"

	"go.uber.org/zap"
)

var (
	ctxPoolNew = expvar.NewInt("ioContext.New")
	ctxPoolPut = expvar.NewInt("ioContext.Put")
)

type IOContextPool struct {
	ctxPool *x.Pool[*IOContext]
	hdrPool *shared.HeadersProxyPool
}

func NewIOContextPool(logger *zap.Logger, hp *shared.HeadersProxyPool, concurrent int64) *IOContextPool {
	ctxp := &IOContextPool{
		hdrPool: hp,
	}
	ctxp.ctxPool = x.NewPool[*IOContext](x.DefaultPoolCapacity).
		WithFactory(func() *IOContext {
			ctxPoolNew.Add(1)
			return newIOContext(logger, concurrent)
		})

	return ctxp
}

func (p *IOContextPool) Get(ctx context.Context) *IOContext {
	return p.GetWithLifetime(ctx, context.Background(), nil)
}

func (p *IOContextPool) GetWithLifetime(ctx, lifetime context.Context, done <-chan struct{}) *IOContext {
	t := p.ctxPool.Get()
	t.extendedCtx, t.extendedCtxCancel = context.WithCancel(context.WithoutCancel(ctx))
	t.reqCtx = ctx
	t.runtimeContext = lifetime
	t.runtimeDone = done
	t.hdrPool = p.hdrPool
	t.shouldExtend.Store(false)
	t.extenders = 0
	t.waiting = false
	t.extensionDone = make(chan struct{})
	// Capture the cancel function, rather than a mutable field on the pooled object.
	cancel := t.extendedCtxCancel
	t.stopRequest = context.AfterFunc(ctx, func() {
		if !t.shouldExtend.Load() {
			cancel()
		}
	})
	t.stopRuntime = context.AfterFunc(lifetime, cancel)
	return t
}

func (p *IOContextPool) Put(t *IOContext) {
	go func() {
		t.wait()
		t.release()
		t.hdrPool = nil
		t.reqCtx = nil
		t.runtimeContext = nil
		t.extendedCtxCancel = nil
		t.extendedCtx = nil
		t.runtimeDone = nil
		t.stopRequest = nil
		t.stopRuntime = nil
		p.ctxPool.Put(t)
		ctxPoolPut.Add(1)
	}()
}

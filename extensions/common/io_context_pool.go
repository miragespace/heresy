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
	t := p.ctxPool.Get()
	t.extendedCtx, t.extendedCtxCancel = context.WithCancel(context.Background())
	t.reqCtx = ctx
	t.hdrPool = p.hdrPool
	t.shouldExtend.Store(false)
	return t
}

func (p *IOContextPool) Put(t *IOContext) {
	go t.wait()
	go func() {
		t.release()
		t.hdrPool = nil
		t.reqCtx = nil
		t.extendedCtxCancel = nil
		t.extendedCtx = nil
		p.ctxPool.Put(t)
		ctxPoolPut.Add(1)
	}()
}

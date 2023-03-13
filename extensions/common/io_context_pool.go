package common

import (
	"context"
	"sync"

	"go.miragespace.co/heresy/extensions/common/shared"

	"go.uber.org/zap"
)

type IOContextPool struct {
	ctxPool sync.Pool
	hdrPool *shared.HeadersProxyPool
}

func NewIOContextPool(logger *zap.Logger, hp *shared.HeadersProxyPool, concurrent int64) *IOContextPool {
	return &IOContextPool{
		ctxPool: sync.Pool{
			New: func() any {
				return newIOContext(logger, concurrent)
			},
		},
		hdrPool: hp,
	}
}

func (p *IOContextPool) Get(ctx context.Context) *IOContext {
	t := p.ctxPool.Get().(*IOContext)
	t.extendedCtx, t.extendedCtxCancel = context.WithCancel(context.Background())
	t.reqCtx = ctx
	t.hdrPool = p.hdrPool
	t.shouldExtend.Store(false)
	return t
}

func (p *IOContextPool) Put(t *IOContext) {
	go func() {
		t.release()
		t.hdrPool = nil
		t.reqCtx = nil
		t.extendedCtxCancel = nil
		t.extendedCtx = nil
		p.ctxPool.Put(t)
	}()
}

package common

import (
	"context"
	"sync"
)

type IOContextPool struct {
	ctxPool sync.Pool
	hdrPool *HeadersProxyPool
}

func NewIOContextPool(concurrent int64) *IOContextPool {
	return &IOContextPool{
		ctxPool: sync.Pool{
			New: func() any {
				return newIOContext(concurrent)
			},
		},
	}
}

func (p *IOContextPool) WithHeadersPool(hp *HeadersProxyPool) {
	p.hdrPool = hp
}

func (p *IOContextPool) Get(ctx context.Context) *IOContext {
	t := p.ctxPool.Get().(*IOContext)
	t.ctx = ctx
	t.hdrPool = p.hdrPool
	return t
}

func (p *IOContextPool) Put(t *IOContext) {
	go func() {
		t.release()
		t.hdrPool = nil
		t.ctx = nil
		p.ctxPool.Put(t)
	}()
}

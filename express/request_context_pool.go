package express

import (
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type RequestContextPool struct {
	ctxPool sync.Pool
	chPool  sync.Pool
}

func NewRequestContextPool(eventLoop *eventloop.EventLoop) *RequestContextPool {
	pool := &RequestContextPool{}
	pool.chPool = sync.Pool{
		New: func() any {
			return make(chan *RequestContext, 1)
		},
	}
	pool.ctxPool = sync.Pool{
		New: func() any {
			ctxCh := pool.chPool.Get().(chan *RequestContext)
			defer pool.chPool.Put(ctxCh)

			// initialization of new native variable has to be
			// ran on the loop
			eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				ctxCh <- newRequestContext(vm)
			})
			return <-ctxCh
		},
	}
	return pool
}

func (p *RequestContextPool) Get() *RequestContext {
	return p.ctxPool.Get().(*RequestContext)
}

func (p *RequestContextPool) Put(ctx *RequestContext) {
	ctx.reset()
	p.ctxPool.Put(ctx)
}

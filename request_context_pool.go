package heresy

import (
	"sync"

	"github.com/dop251/goja"
)

type requestContextPool struct {
	ctxPool sync.Pool
	chPool  sync.Pool
}

func newRequestContextPool(inst *runtimeInstance) *requestContextPool {
	pool := &requestContextPool{}
	pool.chPool = sync.Pool{
		New: func() any {
			return make(chan *requestContext, 1)
		},
	}
	pool.ctxPool = sync.Pool{
		New: func() any {
			ctxCh := pool.chPool.Get().(chan *requestContext)
			defer pool.chPool.Put(ctxCh)

			// initialization of new native variable has to be
			// ran on the loop
			inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				ctxCh <- newRequestContext(vm)
			})
			return <-ctxCh
		},
	}
	return pool
}

func (p *requestContextPool) Get() *requestContext {
	return p.ctxPool.Get().(*requestContext)
}

func (p *requestContextPool) Put(ctx *requestContext) {
	ctx.reset()
	p.ctxPool.Put(ctx)
}

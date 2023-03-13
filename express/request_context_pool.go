package express

import (
	"sync"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/fetch"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"go.uber.org/zap"
)

type RequestContextDeps struct {
	Logger    *zap.Logger
	Eventloop *eventloop.EventLoop
	Fetch     *fetch.Fetch
}

type RequestContextPool struct {
	ctxPool sync.Pool
	chPool  sync.Pool
}

func NewRequestContextPool(deps RequestContextDeps) *RequestContextPool {
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
			deps.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				ctxCh <- newRequestContext(vm, deps)
			})
			return <-ctxCh
		},
	}
	return pool
}

func (p *RequestContextPool) Get(t *common.IOContext) *RequestContext {
	ctx := p.ctxPool.Get().(*RequestContext)
	ctx.ioContext = t
	t.RegisterCleanup(func() {
		p.put(ctx)
	})
	return ctx
}

func (p *RequestContextPool) put(ctx *RequestContext) {
	ctx.reset()
	p.ctxPool.Put(ctx)
}

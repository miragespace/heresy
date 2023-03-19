package express

import (
	"expvar"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/x"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/kv"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"go.uber.org/zap"
)

var (
	ctxNew = expvar.NewInt("requestCtx.New")
	ctxPut = expvar.NewInt("requestCtx.Put")
)

type RequestContextDeps struct {
	Logger    *zap.Logger
	Eventloop *eventloop.EventLoop
	Fetch     *fetch.Fetch
	KV        *kv.KVManager
}

type RequestContextPool struct {
	ctxPool *x.Pool[*RequestContext]
}

func NewRequestContextPool(deps RequestContextDeps) *RequestContextPool {
	pool := &RequestContextPool{}
	pool.ctxPool = x.NewPool[*RequestContext](x.DefaultPoolCapacity).
		WithFactory(func() *RequestContext {
			ctxCh := make(chan *RequestContext, 1)
			// initialization of new native variable has to be
			// ran on the loop
			deps.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				ctxNew.Add(1)
				ctxCh <- newRequestContext(vm, deps)
			})
			return <-ctxCh
		})
	return pool
}

func (p *RequestContextPool) Get(t *common.IOContext) *RequestContext {
	ctx := p.ctxPool.Get()
	ctx.ioContext = t
	t.RegisterCleanup(func() {
		p.put(ctx)
	})
	return ctx
}

func (p *RequestContextPool) put(ctx *RequestContext) {
	ctx.reset()
	p.ctxPool.Put(ctx)
	ctxPut.Add(1)
}

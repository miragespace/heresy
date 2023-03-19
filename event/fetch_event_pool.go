package event

import (
	"expvar"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/x"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/kv"
	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"go.uber.org/zap"
)

var (
	eventNew = expvar.NewInt("fetchEvent.New")
	eventPut = expvar.NewInt("fetchEvent.Put")
)

type FetchEventDeps struct {
	Logger    *zap.Logger
	Symbols   *polyfill.RuntimeSymbols
	Eventloop *eventloop.EventLoop
	Stream    *stream.StreamController
	Resolver  *promise.PromiseResolver
	Fetch     *fetch.Fetch
	KV        *kv.KVManager
}

type FetchEventPool struct {
	evtPool *x.Pool[*FetchEvent]
}

func NewFetchEventPool(deps FetchEventDeps) *FetchEventPool {
	pool := &FetchEventPool{}

	pool.evtPool = x.NewPool[*FetchEvent](x.DefaultPoolCapacity).
		WithFactory(func() *FetchEvent {
			ctxCh := make(chan *FetchEvent, 1)
			// initialization of new native variable has to be
			// ran on the loop
			deps.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				eventNew.Add(1)
				ctxCh <- newFetchEvent(vm, deps)
			})
			return <-ctxCh
		})

	return pool
}

func (p *FetchEventPool) Get(t *common.IOContext) *FetchEvent {
	f := p.evtPool.Get()
	f.ioContext = t
	t.RegisterCleanup(func() {
		p.put(f)
	})
	return f
}

func (p *FetchEventPool) put(evt *FetchEvent) {
	evt.reset()
	p.evtPool.Put(evt)
	eventPut.Add(1)
}

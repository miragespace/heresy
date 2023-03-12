package event

import (
	"sync"

	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"go.uber.org/zap"
)

type FetchEventDeps struct {
	Logger    *zap.Logger
	Symbols   *polyfill.RuntimeSymbols
	Eventloop *eventloop.EventLoop
	Stream    *stream.StreamController
	Resolver  *promise.PromiseResolver
	Fetch     *fetch.Fetch
}

type FetchEventPool struct {
	evtPool sync.Pool
	chPool  sync.Pool
}

func NewFetchEventPool(deps FetchEventDeps) *FetchEventPool {
	pool := &FetchEventPool{}
	pool.chPool = sync.Pool{
		New: func() any {
			return make(chan *FetchEvent, 1)
		},
	}
	pool.evtPool = sync.Pool{
		New: func() any {
			ctxCh := pool.chPool.Get().(chan *FetchEvent)
			defer pool.chPool.Put(ctxCh)

			// initialization of new native variable has to be
			// ran on the loop
			deps.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				ctxCh <- newFetchEvent(vm, deps)
			})
			return <-ctxCh
		},
	}
	return pool
}

func (p *FetchEventPool) Get() *FetchEvent {
	return p.evtPool.Get().(*FetchEvent)
}

func (p *FetchEventPool) Put(evt *FetchEvent) {
	evt.reset()
	p.evtPool.Put(evt)
}

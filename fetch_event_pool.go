package heresy

import (
	"sync"

	"github.com/dop251/goja"
)

type fetchEventPool struct {
	evtPool sync.Pool
	chPool  sync.Pool
}

func newFetchEventPool(inst *runtimeInstance) *fetchEventPool {
	pool := &fetchEventPool{}
	pool.chPool = sync.Pool{
		New: func() any {
			return make(chan *fetchEvent, 1)
		},
	}
	pool.evtPool = sync.Pool{
		New: func() any {
			ctxCh := pool.chPool.Get().(chan *fetchEvent)
			defer pool.chPool.Put(ctxCh)

			// initialization of new native variable has to be
			// ran on the loop
			inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				ctxCh <- newFetchEvent(vm, inst.stream)
			})
			return <-ctxCh
		},
	}
	return pool
}

func (p *fetchEventPool) Get() *fetchEvent {
	return p.evtPool.Get().(*fetchEvent)
}

func (p *fetchEventPool) Put(evt *fetchEvent) {
	evt.Reset()
	p.evtPool.Put(evt)
}

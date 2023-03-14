package shared

import (
	"expvar"

	"go.miragespace.co/heresy/extensions/common/x"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
)

var (
	hdrPxyNew = expvar.NewInt("headersProxy.New")
	hdrPxyPut = expvar.NewInt("headersProxy.Put")
)

type HeadersProxyPool struct {
	hdrPool *x.Pool[*HeadersProxy]
}

func NewHeadersProxyPool(vm *goja.Runtime, symbols *polyfill.RuntimeSymbols) *HeadersProxyPool {
	hp := &HeadersProxyPool{}
	hp.hdrPool = x.NewPool[*HeadersProxy](x.DefaultPoolCapacity).
		WithFactory(func() *HeadersProxy {
			hdrPxyNew.Add(1)
			return newHeadersProxy(vm, symbols)
		})
	return hp
}

func (p *HeadersProxyPool) Get() *HeadersProxy {
	h := p.hdrPool.Get()
	return h
}

func (p *HeadersProxyPool) Put(h *HeadersProxy) {
	h.unsetHeader()
	p.hdrPool.Put(h)
	hdrPxyPut.Add(1)
}

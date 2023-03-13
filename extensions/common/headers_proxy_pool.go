package common

import (
	"sync"

	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
)

type HeadersProxyPool struct {
	hdrPool sync.Pool
}

func NewHeadersProxyPool(vm *goja.Runtime, symbols *polyfill.RuntimeSymbols) *HeadersProxyPool {
	return &HeadersProxyPool{
		hdrPool: sync.Pool{
			New: func() any {
				return newHeadersProxy(vm, symbols)
			},
		},
	}
}

func (p *HeadersProxyPool) Get() *HeadersProxy {
	h := p.hdrPool.Get().(*HeadersProxy)
	return h
}

func (p *HeadersProxyPool) put(h *HeadersProxy) {
	h.unsetHeader()
	p.hdrPool.Put(h)
}

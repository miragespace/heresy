package fetch

import (
	"sync"

	"github.com/dop251/goja"
	"go.miragespace.co/heresy/extensions/stream"
)

type responseProxyPool struct {
	respPool sync.Pool
}

func newResponseProxyPool(vm *goja.Runtime, stream *stream.StreamController) *responseProxyPool {
	return &responseProxyPool{
		respPool: sync.Pool{
			New: func() any {
				return newResponseProxy(vm, stream)
			},
		},
	}
}

func (p *responseProxyPool) Get() *responseProxy {
	return p.respPool.Get().(*responseProxy)
}

func (p *responseProxyPool) Put(resp *responseProxy) {
	p.respPool.Put(resp)
}

package heresy

import (
	"context"
	"net/http"
	"sync"

	"github.com/dop251/goja"
)

type nativeResolverPool struct {
	vm   *goja.Runtime
	pool sync.Pool
}

func newResolverPool(vm *goja.Runtime) *nativeResolverPool {
	return &nativeResolverPool{
		vm: vm,
		pool: sync.Pool{
			New: func() any {
				return newNativeResolver(vm)
			},
		},
	}
}

func (p *nativeResolverPool) Get(w http.ResponseWriter, r *http.Request) *nativeRequestProxy {
	return p.pool.Get().(*nativeRequestProxy).with(w, r)
}

func (p *nativeResolverPool) Put(h *nativeRequestProxy) {
	p.pool.Put(h)
}

func (p *nativeResolverPool) Request(req *http.Request) goja.Value {
	return p.vm.ToValue(req.RequestURI)
}

func (p *nativeResolverPool) Interrupt() {
	p.vm.Interrupt(context.Canceled)
}

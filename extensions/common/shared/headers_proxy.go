package shared

import (
	"fmt"
	"net/http"
	"sort"

	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
)

type HeadersProxy struct {
	nativeObj *goja.Object
	header    http.Header
	vm        *goja.Runtime
}

var _ goja.DynamicObject = (*HeadersProxy)(nil)

func newHeadersProxy(vm *goja.Runtime, symbols *polyfill.RuntimeSymbols) *HeadersProxy {
	headersInstance, err := symbols.Headers()(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Headers) constructor call returned an error: %w", err))
	}

	proxy := &HeadersProxy{
		vm:        vm,
		nativeObj: headersInstance,
	}
	headersInstance.Set("map", vm.NewDynamicObject(proxy))

	return proxy
}

func (h *HeadersProxy) UseHeader(header http.Header) {
	h.header = header
}

func (h *HeadersProxy) unsetHeader() {
	h.header = nil
}

func (h *HeadersProxy) NativeObject() goja.Value {
	return h.nativeObj
}

func (h *HeadersProxy) Get(key string) goja.Value {
	v := h.header.Get(key)
	if v != "" {
		return h.vm.ToValue(v)
	}
	return goja.Undefined()
}

func (h *HeadersProxy) Set(key string, val goja.Value) bool {
	return false
}

func (h *HeadersProxy) Has(key string) bool {
	return !goja.IsUndefined(h.Get(key))
}

func (h *HeadersProxy) Delete(key string) bool {
	return false
}

func (h *HeadersProxy) Keys() []string {
	keys := make([]string, 0)
	for k := range h.header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

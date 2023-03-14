package shared

import (
	"fmt"
	"net/http"
	"sort"

	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
)

type HeadersProxy struct {
	nativeObj        *goja.Object
	header           http.Header
	vm               *goja.Runtime
	nativeProperties map[string]goja.Value
	keys             []string
}

var _ goja.DynamicObject = (*HeadersProxy)(nil)

func newHeadersProxy(vm *goja.Runtime, symbols *polyfill.RuntimeSymbols) *HeadersProxy {
	headersInstance, err := symbols.Headers()(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Headers) constructor call returned an error: %w", err))
	}

	proxy := &HeadersProxy{
		vm:               vm,
		nativeObj:        headersInstance,
		nativeProperties: map[string]goja.Value{},
		keys:             make([]string, 0, 1),
	}
	headersInstance.Set("map", vm.NewDynamicObject(proxy))

	return proxy
}

func (h *HeadersProxy) UseHeader(header http.Header) {
	h.header = header
	for k := range h.header {
		h.keys = append(h.keys, k)
	}
	sort.Strings(h.keys)
}

func (h *HeadersProxy) unsetHeader() {
	for k := range h.nativeProperties {
		delete(h.nativeProperties, k)
	}
	h.keys = h.keys[:0]
	h.header = nil
}

func (h *HeadersProxy) NativeObject() goja.Value {
	return h.nativeObj
}

func (h *HeadersProxy) Get(key string) goja.Value {
	if h.nativeProperties[key] == nil {
		v := h.header.Get(key)
		if v == "" {
			return goja.Undefined()
		}
		h.nativeProperties[key] = h.vm.ToValue(v)
	}
	return h.nativeProperties[key]
}

func (h *HeadersProxy) Set(key string, val goja.Value) bool {
	return false
}

func (h *HeadersProxy) Has(key string) bool {
	for _, k := range h.keys {
		if k == key {
			return true
		}
	}
	return false
}

func (h *HeadersProxy) Delete(key string) bool {
	return false
}

func (h *HeadersProxy) Keys() []string {
	return h.keys
}

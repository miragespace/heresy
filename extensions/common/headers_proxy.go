package common

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/dop251/goja"
)

type HeadersProxy struct {
	Runtime   *goja.Runtime
	nativeObj *goja.Object
	header    http.Header
}

var _ goja.DynamicObject = (*HeadersProxy)(nil)

func NewHeadersProxy(vm *goja.Runtime) *HeadersProxy {
	headesrClass := vm.Get("Headers")
	headersConstructor, ok := goja.AssertConstructor(headesrClass)
	if !ok {
		panic("runtime panic: Headers is not a constructor, please check if polyfill is enabled")
	}

	headersInstance, err := headersConstructor(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Headers) constructor call returned an error: %w", err))
	}

	proxy := &HeadersProxy{
		Runtime:   vm,
		nativeObj: headersInstance,
	}
	headersInstance.Set("map", vm.NewDynamicObject(proxy))

	return proxy
}

func (h *HeadersProxy) UseHeader(header http.Header) {
	h.header = header
}

func (h *HeadersProxy) UnsetHeader() {
	h.header = nil
}

func (h *HeadersProxy) NativeObject() goja.Value {
	return h.nativeObj
}

func (h *HeadersProxy) Get(key string) goja.Value {
	v := h.header.Get(key)
	if v != "" {
		return h.Runtime.ToValue(v)
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

package heresy

import (
	"fmt"
	"sort"

	"github.com/dop251/goja"
)

type fetchEventRequestHeaders struct {
	*fetchEvent
	headersProxy  *headersProxy
	nativeHeaders *goja.Object
}

func newFetchEventRequestHeaders(evt *fetchEvent) *fetchEventRequestHeaders {
	proxy := &headersProxy{
		fetchEvent: evt,
	}
	proxy.nativeObj = evt.vm.NewDynamicObject(proxy)

	headesrClass := evt.vm.Get("Headers")
	headersConstructor, ok := goja.AssertConstructor(headesrClass)
	if !ok {
		panic("runtime panic: Headers is not a constructor, please check if polyfill is enabled")
	}

	headersInstance, err := headersConstructor(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Headers) constructor call returned an error: %w", err))
	}

	headersInstance.Set("map", proxy.nativeObj)

	h := &fetchEventRequestHeaders{
		fetchEvent:   evt,
		headersProxy: proxy,
	}
	h.nativeHeaders = headersInstance

	return h
}

type headersProxy struct {
	*fetchEvent
	nativeObj *goja.Object
}

func (h *headersProxy) Get(key string) goja.Value {
	v := h.httpReq.Header.Get(key)
	if v != "" {
		return h.vm.ToValue(v)
	}
	return goja.Undefined()
}

func (h *headersProxy) Set(key string, val goja.Value) bool {
	return false
}

func (h *headersProxy) Has(key string) bool {
	return !goja.IsUndefined(h.Get(key))
}

func (h *headersProxy) Delete(key string) bool {
	return false
}

func (h *headersProxy) Keys() []string {
	keys := make([]string, 0)
	for k := range h.httpReq.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

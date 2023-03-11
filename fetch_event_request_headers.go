package heresy

import (
	"fmt"

	"github.com/dop251/goja"
)

type fetchEventRequestHeaders struct {
	*fetchEvent
	nativeHeaders *goja.Object
}

func newFetchEventRequestHeaders(evt *fetchEvent) *fetchEventRequestHeaders {
	h := &fetchEventRequestHeaders{
		fetchEvent: evt,
	}

	dynamicHeaderMap := h.vm.NewDynamicObject(h)
	headesrClass := evt.vm.Get("Headers")
	headersConstructor, ok := goja.AssertConstructor(headesrClass)
	if !ok {
		panic("runtime panic: Headers is not a constructor, please check if polyfill is enabled")
	}

	headersInstance, err := headersConstructor(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Headers) constructor call returned an error: %w", err))
	}

	headersInstance.Set("map", dynamicHeaderMap)
	h.nativeHeaders = headersInstance

	return h
}

func (h *fetchEventRequestHeaders) Get(key string) goja.Value {
	v := h.httpReq.Header.Get(key)
	if v != "" {
		return h.vm.ToValue(v)
	}
	return goja.Undefined()
}

func (h *fetchEventRequestHeaders) Set(key string, val goja.Value) bool {
	return false
}

func (h *fetchEventRequestHeaders) Has(key string) bool {
	return !goja.IsUndefined(h.Get(key))
}

func (h *fetchEventRequestHeaders) Delete(key string) bool {
	return false
}

func (h *fetchEventRequestHeaders) Keys() []string {
	keys := make([]string, 0)
	for k := range h.httpReq.Header {
		keys = append(keys, k)
	}
	return keys
}

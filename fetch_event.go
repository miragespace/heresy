package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
	"go.miragespace.co/heresy/extensions/stream"
)

type fetchEvent struct {
	httpReq       *http.Request
	httpResp      http.ResponseWriter
	httpNext      http.Handler
	requestProxy  *fetchEventRequest
	nativeResolve goja.Value
	nativeReject  goja.Value
	done          chan struct{}
	vm            *goja.Runtime
	nativeEvt     *goja.Object
	skipNext      bool
}

var _ goja.DynamicObject = (*fetchEvent)(nil)

func newFetchEvent(vm *goja.Runtime, controller *stream.StreamController) *fetchEvent {
	evt := &fetchEvent{
		done: make(chan struct{}, 1),
		vm:   vm,
	}

	return evt
}

func (evt *fetchEvent) init(vm *goja.Runtime, controller *stream.StreamController) {
	evt.requestProxy = newFetchEventRequest(evt, controller)
	evt.nativeResolve = evt.getNativeEventResolver()
	evt.nativeReject = evt.getNativeEventRejector()
	evt.nativeEvt = vm.NewDynamicObject(evt)
}

func (evt *fetchEvent) reset() {
	evt.httpReq = nil
	evt.httpResp = nil
	evt.httpNext = nil
	evt.skipNext = false
}

func (evt *fetchEvent) Get(key string) goja.Value {
	switch key {
	case "respondWith":
		return evt.vm.ToValue(evt.nativeRespondWith)
	case "waitUntil":
		return evt.vm.ToValue(evt.nativeWaitUntil)
	case "request":
		return evt.requestProxy.nativeReq
	default:
		return goja.Undefined()
	}
}

func (evt *fetchEvent) Set(key string, val goja.Value) bool {
	return false
}

func (evt *fetchEvent) Has(key string) bool {
	return !goja.IsUndefined(evt.Get(key))
}

func (evt *fetchEvent) Delete(key string) bool {
	return false
}

func (evt *fetchEvent) Keys() []string {
	return []string{"request", "respondWith", "waitUntil"}
}

func (evt *fetchEvent) WithHttp(w http.ResponseWriter, r *http.Request, next http.Handler) *fetchEvent {
	evt.httpResp = w
	evt.httpReq = r
	evt.httpNext = next

	return evt
}

func (evt *fetchEvent) nativeRespondWith(fc goja.FunctionCall) goja.Value {
	evt.skipNext = true
	return goja.Undefined()
}

func (evt *fetchEvent) nativeWaitUntil(fc goja.FunctionCall) goja.Value {
	return goja.Undefined()
}

func (evt *fetchEvent) wait() {
	<-evt.done
}

func (evt *fetchEvent) exception(err error) {
	select {
	case <-evt.httpReq.Context().Done():
	default:
		evt.httpResp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(evt.httpResp, "Unexpected runtime exception: %+v", err)
	}
	evt.done <- struct{}{}
}

func (evt *fetchEvent) getNativeEventResolver() goja.Value {
	return nativeEventWrapper(evt, func(w http.ResponseWriter, r *http.Request, _ goja.Value) {
		if evt.skipNext {
			return
		}
		evt.httpNext.ServeHTTP(w, r)
	})
}

func (evt *fetchEvent) getNativeEventRejector() goja.Value {
	return nativeEventWrapper(evt, func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		w.WriteHeader(http.StatusInternalServerError)
		if goja.IsUndefined(v) {
			return
		}
		fmt.Fprintf(w, "Execution exception: %+v", v)
	})
}

func nativeEventWrapper(
	evt *fetchEvent,
	fn func(w http.ResponseWriter, r *http.Request, v goja.Value),
) goja.Value {
	return evt.vm.ToValue(func(fc goja.FunctionCall) goja.Value {
		select {
		case <-evt.httpReq.Context().Done():
		default:
			v := fc.Argument(0)
			fn(evt.httpResp, evt.httpReq, v)
		}
		evt.done <- struct{}{}
		return goja.Undefined()
	})
}

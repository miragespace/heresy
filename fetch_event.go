package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
	"go.miragespace.co/heresy/extensions/stream"
)

type fetchEvent struct {
	stream            *stream.StreamController
	httpReq           *http.Request
	httpResp          http.ResponseWriter
	httpNext          http.Handler
	requestProxy      *fetchEventRequest
	nativeFetch       goja.Value
	nativeResolve     goja.Value
	nativeReject      goja.Value
	done              chan struct{}
	vm                *goja.Runtime
	nativeEvt         *goja.Object
	nativeEvtInstance *goja.Object
	hasFetch          bool
	skipNext          bool
}

var _ goja.DynamicObject = (*fetchEvent)(nil)

func newFetchEvent(vm *goja.Runtime, controller *stream.StreamController) *fetchEvent {
	evt := &fetchEvent{
		stream: controller,
		done:   make(chan struct{}, 1),
		vm:     vm,
	}

	fetchEventClass := evt.vm.Get("FetchEvent")
	fetchEventConstructor, ok := goja.AssertConstructor(fetchEventClass)
	if !ok {
		panic("runtime panic: FetchEvent is not a constructor, please check if polyfill is enabled")
	}

	var err error
	evt.nativeEvtInstance, err = fetchEventConstructor(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new FetchEvent) constructor call returned an error: %w", err))
	}

	evt.requestProxy = newFetchEventRequest(evt)
	evt.nativeResolve = evt.getNativeEventResolver()
	evt.nativeReject = evt.getNativeEventRejector()
	evt.nativeEvt = evt.vm.NewDynamicObject(evt)
	evt.nativeEvt.SetPrototype(evt.nativeEvtInstance.Prototype())

	return evt
}

func (evt *fetchEvent) Reset() {
	evt.httpReq = nil
	evt.httpResp = nil
	evt.httpNext = nil
	evt.nativeFetch = nil
	evt.hasFetch = false
	evt.skipNext = false
	evt.requestProxy.Reset()
}

func (evt *fetchEvent) Get(key string) goja.Value {
	switch key {
	case "respondWith":
		return evt.vm.ToValue(evt.nativeRespondWith)
	case "waitUntil":
		return evt.vm.ToValue(evt.nativeWaitUntil)
	case "request":
		return evt.requestProxy.nativeReq
	case "fetch":
		if evt.hasFetch {
			return evt.nativeFetch
		}
		fallthrough
	default:
		return evt.nativeEvtInstance.Get(key)
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
	return []string{"request"}
}

func (evt *fetchEvent) WithHttp(w http.ResponseWriter, r *http.Request, next http.Handler) *fetchEvent {
	evt.httpResp = w
	evt.httpReq = r
	evt.httpNext = next

	evt.requestProxy.initialize()

	return evt
}

func (evt *fetchEvent) WithFetch(f goja.Value) *fetchEvent {
	evt.hasFetch = true
	evt.nativeFetch = f
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
	return evt.nativeEventWrapper(func(w http.ResponseWriter, r *http.Request, _ goja.Value) {
		if evt.skipNext {
			return
		}
		evt.httpNext.ServeHTTP(w, r)
	})
}

func (evt *fetchEvent) getNativeEventRejector() goja.Value {
	return evt.nativeEventWrapper(func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		w.WriteHeader(http.StatusInternalServerError)
		if goja.IsUndefined(v) {
			return
		}
		fmt.Fprintf(w, "Execution exception: %+v", v)
	})
}

func (evt *fetchEvent) nativeEventWrapper(
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

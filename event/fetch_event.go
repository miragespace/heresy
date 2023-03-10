package event

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
	pool "github.com/libp2p/go-buffer-pool"
	"go.uber.org/zap"
)

type FetchEvent struct {
	httpReq               *http.Request
	httpResp              http.ResponseWriter
	httpNext              http.Handler
	ioContext             *common.IOContext
	requestProxy          *fetchEventRequest
	nativeFetch           goja.Value
	nativeRequestResolve  goja.Value
	nativeRequestReject   goja.Value
	nativeResponseResolve goja.Value
	nativeResponseReject  goja.Value
	nativeConclude        goja.Value
	requestDone           chan struct{}
	responseDone          chan struct{}
	deps                  FetchEventDeps
	vm                    *goja.Runtime
	nativeEvt             *goja.Object
	nativeEvtInstance     *goja.Object
	hasFetch              bool
	skipNext              bool
	useRespondWith        bool
	responseSent          bool
}

var _ goja.DynamicObject = (*FetchEvent)(nil)

func newFetchEvent(vm *goja.Runtime, deps FetchEventDeps) *FetchEvent {
	evt := &FetchEvent{
		nativeEvtInstance: deps.Symbols.FetchEvent(),
		requestDone:       make(chan struct{}, 1),
		responseDone:      make(chan struct{}, 1),
		deps:              deps,
		vm:                vm,
	}

	evt.requestProxy = newFetchEventRequest(evt)
	evt.nativeRequestResolve = evt.getNativeRequestResolver()
	evt.nativeRequestReject = evt.getNativeRequestRejector()
	evt.nativeResponseResolve = evt.getNativeResponseResolver()
	evt.nativeResponseReject = evt.getNativeResponseRejector()
	evt.nativeEvt = evt.vm.NewDynamicObject(evt)
	evt.nativeEvt.SetPrototype(evt.nativeEvtInstance.Prototype())

	return evt
}

func (evt *FetchEvent) reset() {
	evt.httpReq = nil
	evt.httpResp = nil
	evt.httpNext = nil
	evt.nativeFetch = nil
	evt.nativeConclude = nil
	evt.hasFetch = false
	evt.skipNext = false
	evt.useRespondWith = false
	evt.responseSent = false
	evt.requestProxy.reset()
	evt.ioContext = nil
}

func (evt *FetchEvent) Get(key string) goja.Value {
	switch key {
	case "respondWith":
		return evt.vm.ToValue(evt.nativeRespondWith)
	case "waitUntil":
		return evt.vm.ToValue(evt.nativeWaitUntil)
	case "request":
		return evt.requestProxy.nativeReq
	case "fetch":
		if evt.hasFetch {
			// lazy initialization
			if evt.nativeFetch == nil {
				fetcher := evt.deps.Fetch.NewNativeFetchVM(evt.ioContext, evt.vm)
				evt.nativeFetch = fetcher.NativeFunc()
			}
			return evt.nativeFetch
		}
		fallthrough
	default:
		return evt.nativeEvtInstance.Get(key)
	}
}

func (evt *FetchEvent) Set(key string, val goja.Value) bool {
	return false
}

func (evt *FetchEvent) Has(key string) bool {
	return !goja.IsUndefined(evt.Get(key))
}

func (evt *FetchEvent) Delete(key string) bool {
	return false
}

func (evt *FetchEvent) Keys() []string {
	return []string{"request"}
}

func (evt *FetchEvent) WithHttp(w http.ResponseWriter, r *http.Request, next http.Handler) *FetchEvent {
	evt.httpResp = w
	evt.httpReq = r
	evt.httpNext = next

	return evt
}

func (evt *FetchEvent) EnableFetch() {
	evt.hasFetch = true
}

func (evt *FetchEvent) nativeRespondWith(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	resp := fc.Argument(0)
	if goja.IsUndefined(resp) {
		panic(vm.NewTypeError("respondWith: expecting 1 argument, got 0 argument"))
	}

	if evt.useRespondWith {
		panic(vm.NewTypeError("respondWith: already called"))
	}

	evt.skipNext = true
	evt.useRespondWith = true

	if err := evt.deps.Resolver.NewPromiseFuncWithArgVM(
		vm,
		evt.deps.Fetch.GetResponseHelper(),
		resp,
		evt.nativeResponseResolve,
		evt.nativeResponseReject,
	); err != nil {
		panic(vm.NewGoError(fmt.Errorf("runtime panic: failed to register respondWith resolver: %w", err)))
	}

	return goja.Undefined()
}

func (evt *FetchEvent) nativeWaitUntil(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	ret = goja.Undefined()

	promise := fc.Argument(0)
	if goja.IsUndefined(promise) {
		panic(vm.NewTypeError("wailUntil: expecting 1 argument, got 0 argument"))
	}
	if _, ok := promise.Export().(*goja.Promise); !ok {
		panic(vm.NewTypeError("waitUntil: expecting argument as a Promise"))
	}

	evt.ioContext.ExtendContext()
	if evt.nativeConclude == nil {
		evt.nativeConclude = vm.ToValue(func(goja.FunctionCall) goja.Value {
			evt.ioContext.ConcludeExtend()
			return goja.Undefined()
		})
	}
	if err := evt.deps.Resolver.NewPromiseResultVM(
		vm,
		promise,
		evt.nativeConclude,
		evt.nativeConclude,
	); err != nil {
		panic(vm.NewGoError(fmt.Errorf("runtime panic: failed to register waitUntil resolver: %w", err)))
	}

	return
}

func (evt *FetchEvent) Wait() {
	<-evt.requestDone
}

func (evt *FetchEvent) NativeObject() goja.Value {
	return evt.nativeEvt
}

func (evt *FetchEvent) Resolve() goja.Value {
	return evt.nativeRequestResolve
}

func (evt *FetchEvent) Reject() goja.Value {
	return evt.nativeRequestReject
}

func (evt *FetchEvent) Exception(err error) {
	if evt.responseSent {
		return
	}
	select {
	case <-evt.httpReq.Context().Done():
	default:
		evt.httpResp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(evt.httpResp, "Unexpected runtime exception: %+v", err)
	}
	evt.responseSent = true
	evt.wake()
}

func (evt *FetchEvent) getNativeResponseResolver() goja.Value {
	return evt.nativeFunctionWrapper(func(w http.ResponseWriter, r *http.Request, fc goja.FunctionCall) {
		// NOTE: in the nested response resolver/rejector from .respondWith, we do not .wake()
		// to unblock the http request in progress. Resolution should be done by the outer request resolver

		var (
			nativeResp = fc.Argument(0).ToObject(evt.vm)
		)

		if respOk := nativeResp.Get("ok"); !respOk.ToBoolean() {
			// .respondWith did not resolve to a Response (e.g. undefined)
			w.WriteHeader(http.StatusNoContent)
			evt.responseSent = true
			evt.responseDone <- struct{}{}
			return
		}

		var (
			respStatus                 = nativeResp.Get("status")
			respHeaders                = nativeResp.Get("headers")
			respBody                   = nativeResp.Get("body")
			bodyType                   = respBody.ExportType()
			status      int64          = respStatus.ToInteger()
			headers     map[string]any = respHeaders.Export().(map[string]any)
			useBody     io.Reader      = nil
		)

		if goja.IsUndefined(respBody) || goja.IsNull(respBody) {
			// no body
			useBody = http.NoBody
		} else if bodyType.Kind() == reflect.String {
			strBuf := pool.NewBufferString(respBody.String())
			evt.ioContext.RegisterCleanup(strBuf.Reset)
			useBody = strBuf
		} else {
			// possibly wrapped ReadableStream
			reader, ok := stream.AssertReader(respBody, evt.vm)
			if !ok {
				panic(evt.vm.NewGoError(fetch.ErrUnsupportedReadableStream))
			}
			useBody = reader
		}

		go func() {
			for k, v := range headers {
				w.Header().Set(k, fmt.Sprintf("%s", v))
			}

			buf := common.GetBuffer()
			defer common.PutBuffer(buf)

			w.WriteHeader(int(status))
			_, err := io.CopyBuffer(w, useBody, buf)
			if err != nil {
				evt.deps.Logger.Error("Error writing response", zap.Error(err))
			}

			evt.responseSent = true
			evt.responseDone <- struct{}{}
		}()
	})
}
func (evt *FetchEvent) getNativeResponseRejector() goja.Value {
	return evt.nativeFunctionWrapper(func(w http.ResponseWriter, r *http.Request, fc goja.FunctionCall) {
		// NOTE: in the nested response resolver/rejector from .respondWith, we do not .wake()
		// to unblock the http request in progress. Resolution should be done by the outer request resolver

		v := fc.Argument(0)
		w.WriteHeader(http.StatusInternalServerError)
		if goja.IsUndefined(v) {
			return
		}
		fmt.Fprintf(w, "Execution exception: %+v", v)
		evt.responseSent = true
		evt.responseDone <- struct{}{}
	})
}

func (evt *FetchEvent) getNativeRequestResolver() goja.Value {
	return evt.nativeFunctionWrapper(func(w http.ResponseWriter, r *http.Request, _ goja.FunctionCall) {
		go func() {
			defer evt.wake()

			if evt.skipNext {
				// .respondWith was used
				select {
				case <-r.Context().Done():
					return
				case <-evt.responseDone:
				}
			} else {
				// fallthrough, .respondWith did not call
				evt.httpNext.ServeHTTP(w, r)
			}
			evt.responseSent = true
		}()
	})
}

func (evt *FetchEvent) getNativeRequestRejector() goja.Value {
	return evt.nativeFunctionWrapper(func(w http.ResponseWriter, r *http.Request, fc goja.FunctionCall) {
		v := fc.Argument(0)

		go func() {
			defer evt.wake()

			if evt.skipNext {
				// .respondWith was used, but exception thrown
				select {
				case <-r.Context().Done():
					return
				case <-evt.responseDone:
				}
			}

			if evt.responseSent {
				evt.deps.Logger.Warn("Handler thrown exception after response was sent", zap.String("exception", v.String()))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Execution exception: %+v", v)
			evt.responseSent = true
		}()
	})
}

func (evt *FetchEvent) nativeFunctionWrapper(
	fn func(w http.ResponseWriter, r *http.Request, fc goja.FunctionCall),
) goja.Value {
	return evt.vm.ToValue(func(fc goja.FunctionCall) (ret goja.Value) {
		fn(evt.httpResp, evt.httpReq, fc)
		return goja.Undefined()
	})
}

func (evt *FetchEvent) wake() {
	evt.requestDone <- struct{}{}
}

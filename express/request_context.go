package express

import (
	"fmt"
	"net/http"

	"go.miragespace.co/heresy/extensions/common"

	"github.com/dop251/goja"
)

type RequestContext struct {
	httpReq       *http.Request
	httpResp      http.ResponseWriter
	httpNext      http.Handler
	ioContext     *common.IOContext
	responseProxy *contextResponse
	requestProxy  *contextRequest
	nativeFetch   goja.Value
	nativeResolve goja.Value
	nativeReject  goja.Value
	requestDone   chan struct{}
	deps          RequestContextDeps
	vm            *goja.Runtime
	nativeCtx     *goja.Object
	hasFetch      bool
	nextInvoked   bool
	responseSent  bool

	statusSet bool
}

var _ goja.DynamicObject = (*RequestContext)(nil)

func newRequestContext(vm *goja.Runtime, deps RequestContextDeps) *RequestContext {
	ctx := &RequestContext{
		requestDone: make(chan struct{}, 1),
		deps:        deps,
		vm:          vm,
	}
	ctx.nativeResolve = ctx.getNativeContextResolver()
	ctx.nativeReject = ctx.getNativeContextRejector()
	ctx.responseProxy = newContextResponse(ctx)
	ctx.requestProxy = newContextRequest(ctx)
	ctx.nativeCtx = vm.NewDynamicObject(ctx)

	return ctx
}

func (ctx *RequestContext) reset() {
	ctx.httpReq = nil
	ctx.httpResp = nil
	ctx.httpNext = nil
	ctx.nativeFetch = nil
	ctx.hasFetch = false
	ctx.nextInvoked = false
	ctx.responseSent = false
	ctx.responseProxy.reset()
	ctx.ioContext = nil
}

func (ctx *RequestContext) Get(key string) goja.Value {
	switch key {
	case "res":
		return ctx.responseProxy.nativeRes
	case "req":
		return ctx.requestProxy.nativeReq
	case "next":
		return ctx.vm.ToValue(ctx.nativeNext)
	case "fetch":
		if ctx.hasFetch {
			// lazy initialization
			if ctx.nativeFetch == nil {
				fetcher := ctx.deps.Fetch.NewNativeFetchVM(ctx.ioContext, ctx.vm)
				ctx.nativeFetch = fetcher.NativeFunc()
			}
			return ctx.nativeFetch
		}
		fallthrough
	default:
		return goja.Undefined()
	}
}

func (ctx *RequestContext) Set(_ string, _ goja.Value) bool {
	return false
}

func (ctx *RequestContext) Has(key string) bool {
	return !goja.IsUndefined(ctx.Get(key))
}

func (ctx *RequestContext) Delete(key string) bool {
	return false
}

func (ctx *RequestContext) Keys() []string {
	if ctx.hasFetch {
		return []string{"fetch", "next", "req", "res"}
	} else {
		return []string{"next", "req", "res"}
	}
}

func (ctx *RequestContext) WithHttp(w http.ResponseWriter, r *http.Request, next http.Handler) *RequestContext {
	ctx.httpResp = w
	ctx.httpReq = r
	ctx.httpNext = next

	return ctx
}

func (ctx *RequestContext) EnableFetch() {
	ctx.hasFetch = true
}

func (ctx *RequestContext) nativeNext(fc goja.FunctionCall) goja.Value {
	if ctx.nextInvoked {
		return goja.Undefined()
	}
	ctx.nextInvoked = true
	ctx.responseSent = true

	ctx.httpNext.ServeHTTP(ctx.httpResp, ctx.httpReq)
	return goja.Undefined()
}

func (ctx *RequestContext) Wait() {
	<-ctx.requestDone
}

func (ctx *RequestContext) NativeObject() goja.Value {
	return ctx.nativeCtx
}

func (ctx *RequestContext) Resolve() goja.Value {
	return ctx.nativeResolve
}

func (ctx *RequestContext) Reject() goja.Value {
	return ctx.nativeReject
}

func (ctx *RequestContext) Exception(err error) {
	select {
	case <-ctx.httpReq.Context().Done():
	default:
		ctx.httpResp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(ctx.httpResp, "Unexpected runtime exception: %+v", err)
	}
	ctx.responseSent = true
	ctx.wake()
}

func (ctx *RequestContext) getNativeContextResolver() goja.Value {
	return ctx.nativeContextWrapper(func(w http.ResponseWriter, r *http.Request, _ goja.Value) {
		if ctx.statusSet || ctx.responseSent {
			return
		}
		w.WriteHeader(ctx.responseProxy.statusCode)
	})
}

func (ctx *RequestContext) getNativeContextRejector() goja.Value {
	return ctx.nativeContextWrapper(func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		w.WriteHeader(http.StatusInternalServerError)
		if goja.IsUndefined(v) {
			return
		}
		fmt.Fprintf(w, "Execution exception: %+v", v)
	})
}

func (ctx *RequestContext) nativeContextWrapper(
	fn func(w http.ResponseWriter, r *http.Request, v goja.Value),
) goja.Value {
	return ctx.vm.ToValue(func(fc goja.FunctionCall) goja.Value {
		if ctx.nextInvoked || ctx.responseSent {
			ctx.wake()
			return goja.Undefined()
		}
		select {
		case <-ctx.httpReq.Context().Done():
		default:
			v := fc.Argument(0)
			fn(ctx.httpResp, ctx.httpReq, v)
		}
		ctx.responseSent = true
		ctx.wake()
		return goja.Undefined()
	})
}

func (ctx *RequestContext) wake() {
	ctx.requestDone <- struct{}{}
}

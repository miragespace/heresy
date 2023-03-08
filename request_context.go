package heresy

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/dop251/goja"
)

type requestContext struct {
	httpReq       atomic.Value // *http.Request
	httpResp      atomic.Value // http.ResponseWrite
	httpNext      atomic.Value // http.Handler
	responseProxy *contextResponse
	requestProxy  *contextRequest
	nativeFetch   goja.Value
	nativeResolve goja.Value
	nativeReject  goja.Value
	done          chan struct{}
	vm            *goja.Runtime
	nativeCtx     *goja.Object
	hasFetch      bool
	nextInvoked   bool
	responseSent  bool

	statusSet  bool
	statusCode int
}

var _ goja.DynamicObject = (*requestContext)(nil)

func newRequestContext(vm *goja.Runtime) *requestContext {
	ctx := &requestContext{
		done: make(chan struct{}, 1),
		vm:   vm,
	}
	ctx.nativeResolve = ctx.getNativeContextResolver()
	ctx.nativeReject = ctx.getNativeContextRejector()
	ctx.responseProxy = newContextResponse(ctx)
	ctx.requestProxy = newContextRequest(ctx)
	ctx.nativeCtx = vm.NewDynamicObject(ctx)

	return ctx
}

func (ctx *requestContext) reset() {
	ctx.hasFetch = false
	ctx.nextInvoked = false
	ctx.responseSent = false
	ctx.responseProxy.reset()
}

func (ctx *requestContext) Get(key string) goja.Value {
	switch key {
	case "res":
		return ctx.responseProxy.nativeRes
	case "req":
		return ctx.requestProxy.nativeReq
	case "next":
		return ctx.vm.ToValue(ctx.nativeNext)
	case "fetch":
		if ctx.hasFetch {
			return ctx.nativeFetch
		}
		fallthrough
	default:
		return goja.Undefined()
	}
}

func (ctx *requestContext) Set(_ string, _ goja.Value) bool {
	return false
}

func (ctx *requestContext) Has(key string) bool {
	return !goja.IsUndefined(ctx.Get(key))
}

func (ctx *requestContext) Delete(key string) bool {
	return false
}

func (ctx *requestContext) Keys() []string {
	if ctx.hasFetch {
		return []string{"req", "res", "next", "fetch"}
	} else {

		return []string{"req", "res", "next"}
	}
}

func (ctx *requestContext) WithHttp(w http.ResponseWriter, r *http.Request, next http.Handler) *requestContext {
	ctx.httpResp.Store(w)
	ctx.httpReq.Store(r)
	ctx.httpNext.Store(next)

	return ctx
}

func (ctx *requestContext) WithFetch(f *fetchConfig) *requestContext {
	ctx.hasFetch = true
	ctx.nativeFetch = ctx.vm.ToValue(f.nativeFetch)
	return ctx
}

func (ctx *requestContext) nativeNext(fc goja.FunctionCall) goja.Value {
	if ctx.nextInvoked {
		return goja.Undefined()
	}
	ctx.nextInvoked = true
	ctx.responseSent = true

	req := ctx.httpReq.Load().(*http.Request)
	resp := ctx.httpResp.Load().(http.ResponseWriter)
	next := ctx.httpNext.Load().(http.Handler)
	next.ServeHTTP(resp, req)
	return goja.Undefined()
}

func (ctx *requestContext) wait() {
	<-ctx.done
}

func (ctx *requestContext) exception(err error) {
	req := ctx.httpReq.Load().(*http.Request)
	resp := ctx.httpResp.Load().(http.ResponseWriter)
	select {
	case <-req.Context().Done():
	default:
		resp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(resp, "Unexpected runtime exception: %+v", err)
	}
	ctx.responseSent = true
	ctx.done <- struct{}{}
}

func (ctx *requestContext) getNativeContextResolver() goja.Value {
	return nativeContextWrapper(ctx.vm, ctx, func(w http.ResponseWriter, r *http.Request, _ goja.Value) {
		if ctx.statusSet || ctx.responseSent {
			return
		}
		w.WriteHeader(ctx.responseProxy.statusCode)
	})
}

func (ctx *requestContext) getNativeContextRejector() goja.Value {
	return nativeContextWrapper(ctx.vm, ctx, func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		w.WriteHeader(http.StatusInternalServerError)
		if goja.IsUndefined(v) {
			return
		}
		fmt.Fprintf(w, "Execution exception: %+v", v)
	})
}

func nativeContextWrapper(
	vm *goja.Runtime,
	ctx *requestContext,
	fn func(w http.ResponseWriter, r *http.Request, v goja.Value),
) goja.Value {
	return vm.ToValue(func(fc goja.FunctionCall) goja.Value {
		if ctx.nextInvoked || ctx.responseSent {
			ctx.done <- struct{}{}
			return goja.Undefined()
		}
		req := ctx.httpReq.Load().(*http.Request)
		resp := ctx.httpResp.Load().(http.ResponseWriter)
		select {
		case <-req.Context().Done():
		default:
			v := fc.Argument(0)
			fn(resp, req, v)
		}
		ctx.responseSent = true
		ctx.done <- struct{}{}
		return goja.Undefined()
	})
}

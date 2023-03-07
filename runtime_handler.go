package heresy

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/dop251/goja"
)

var (
	ErrRuntimeNotReady = fmt.Errorf("runtime is not ready")
	ErrNoHandler       = fmt.Errorf("script has no http handler configured")
)

func (rt *Runtime) Handler(w http.ResponseWriter, r *http.Request) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	instance := rt.instance

	if instance == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, ErrRuntimeNotReady)
		return
	}

	httpHandler, ok := instance.httpHandler.Load().(goja.Value)
	if !ok {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, ErrNoHandler)
		return
	}

	h := instance.nativePool.Get(w, r)
	defer instance.nativePool.Put(h)

	nativeReq := instance.nativePool.Request(r)
	instance.eventLoop.RunOnLoop(func(*goja.Runtime) {
		if _, err := instance.runtimeResolver(
			goja.Undefined(),
			httpHandler,
			nativeReq,
			h.Resolve(),
			h.Reject(),
		); err != nil {
			h.Exception(err)
			return
		}
	})

	h.Wait()
}

type nativeRequestProxy struct {
	done          chan struct{}
	httpReq       atomic.Value // *http.Request
	httpResp      atomic.Value // http.ResponseWriter
	nativeResolve goja.Value
	nativeReject  goja.Value
}

func newNativeResolver(vm *goja.Runtime) *nativeRequestProxy {
	h := &nativeRequestProxy{
		done: make(chan struct{}, 1),
	}
	h.nativeResolve = getNativeResolver(vm, h)
	h.nativeReject = getNativeRejector(vm, h)
	return h
}

func (n *nativeRequestProxy) with(w http.ResponseWriter, r *http.Request) *nativeRequestProxy {
	n.httpResp.Store(w)
	n.httpReq.Store(r)
	return n
}

func (n *nativeRequestProxy) Resolve() goja.Value {
	return n.nativeResolve
}

func (n *nativeRequestProxy) Reject() goja.Value {
	return n.nativeReject
}

func (n *nativeRequestProxy) Exception(err error) {
	req := n.httpReq.Load().(*http.Request)
	resp := n.httpResp.Load().(http.ResponseWriter)
	select {
	case <-req.Context().Done():
	default:
		resp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(resp, "Unhandled exception: %+v", err)
	}
	n.done <- struct{}{}
}

func (h *nativeRequestProxy) Wait() {
	<-h.done
}

func nativeWrapper(
	vm *goja.Runtime,
	h *nativeRequestProxy,
	fn func(w http.ResponseWriter, r *http.Request, v goja.Value),
) goja.Value {
	return vm.ToValue(func(fc goja.FunctionCall) goja.Value {
		req := h.httpReq.Load().(*http.Request)
		resp := h.httpResp.Load().(http.ResponseWriter)
		select {
		case <-req.Context().Done():
		default:
			v := fc.Argument(0)
			if goja.Undefined().Equals(v) {
				break
			}
			fn(resp, req, v)
		}
		h.done <- struct{}{}
		return goja.Undefined()
	})
}

func getNativeResolver(vm *goja.Runtime, h *nativeRequestProxy) goja.Value {
	return nativeWrapper(vm, h, func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		fmt.Fprint(w, v)
	})
}

func getNativeRejector(vm *goja.Runtime, h *nativeRequestProxy) goja.Value {
	return nativeWrapper(vm, h, func(w http.ResponseWriter, r *http.Request, v goja.Value) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Execution exception: %+v", v)
	})
}

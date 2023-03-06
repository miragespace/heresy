package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

func (rt *Runtime) Handle(w http.ResponseWriter, r *http.Request) {
	runtimeResolver, resolverReady := rt.runtimeResolver.Load().(goja.Callable)
	httpHandler, handlerReady := rt.httpHandler.Load().(goja.Value)

	if !handlerReady || !resolverReady {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Runtime is not ready")
		return
	}

	httpResolver := getRequestResolver(w, r)

	rt.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		if _, err := runtimeResolver(
			goja.Undefined(),
			httpHandler,
			vm.ToValue(r.RequestURI),
			vm.ToValue(httpResolver.NativeResolve(vm)),
			vm.ToValue(httpResolver.NativeReject(vm)),
		); err != nil {
			httpResolver.Exception(err)
			return
		}
	})

	httpResolver.Wait(rt.eventLoop)
}

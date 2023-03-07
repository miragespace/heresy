package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

func (rt *Runtime) Handler(w http.ResponseWriter, r *http.Request) {
	instance := rt.instance.Load()

	if !instance.running.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Runtime is not ready")
		return
	}

	httpHandler, ok := instance.httpHandler.Load().(goja.Value)
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Script has no http handler configured")
		return
	}

	httpResolver := getRequestResolver(w, r)
	instance.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		if _, err := instance.runtimeResolver(
			goja.Undefined(),
			httpHandler,
			vm.ToValue(r.RequestURI),
			vm.ToValue(httpResolver.nativeResolveCallback(vm, rt.scheduler)),
			vm.ToValue(httpResolver.nativeRejectCallback(vm, rt.scheduler)),
		); err != nil {
			httpResolver.exceptionCallback(err, rt.scheduler)
			return
		}
	})

	httpResolver.Wait()
}

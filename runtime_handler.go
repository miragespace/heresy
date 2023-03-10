package heresy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dop251/goja"
	"go.uber.org/zap"
)

var (
	ErrRuntimeNotReady = fmt.Errorf("middleware runtime is not ready")
	ErrNoHandler       = fmt.Errorf("middleware script has no http handler configured")
)

func (rt *Runtime) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt.mu.RLock()
		defer rt.mu.RUnlock()

		instance := rt.instance

		if instance == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, ErrRuntimeNotReady)
			return
		}

		middlewareHandler, ok := instance.middlewareHandler.Load().(goja.Value)
		if !ok {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprint(w, ErrNoHandler)
			return
		}

		ctx := instance.contextPool.Get()
		defer instance.contextPool.Put(ctx)

		ctx = ctx.WithHttp(w, r, next)

		handlerOption := instance.handlerOption.Load()
		if handlerOption.EnableFetch {
			fetch := &fetchConfig{
				context:   r.Context(),
				eventLoop: instance.eventLoop,
				scheduler: rt.scheduler,
				client: &http.Client{
					Timeout: time.Second * 15,
				},
			}
			ctx.WithFetch(fetch)
		}

		if err := instance.resolver.NewPromise(
			middlewareHandler,
			ctx.nativeCtx,
			ctx.nativeResolve,
			ctx.nativeReject,
		); err != nil {
			rt.logger.Error("Unexpected runtime exception", zap.Error(err))
			ctx.exception(err)
		}

		ctx.wait()
	})
}

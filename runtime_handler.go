package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

var (
	ErrRuntimeNotReady     = fmt.Errorf("middleware runtime is not ready")
	ErrNoMiddlewareHandler = fmt.Errorf("middleware script has no handler configured")
)

func (rt *Runtime) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt.shardRun(func(instance *runtimeInstance) {
			if instance == nilInstance {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, ErrRuntimeNotReady)
				return
			}

			middlewareType := instance.middlewareType.Load().(handlerType)
			if middlewareType == handlerTypeUnset {
				w.WriteHeader(http.StatusBadGateway)
				fmt.Fprint(w, ErrNoMiddlewareHandler)
				return
			}

			switch middlewareType {
			case handlerTypeEvent:
				instance.handleAsEvent(w, r, next)
			case handlerTypeExpress:
				instance.handleAsExpress(w, r, next)
			}
		})
	})
}

func (inst *runtimeInstance) handleAsExpress(w http.ResponseWriter, r *http.Request, next http.Handler) {
	middlewareHandler := inst.middlewareHandler.Load().(goja.Value)

	ctx := inst.contextPool.Get()
	defer inst.contextPool.Put(ctx)

	ctx.WithHttp(w, r, next)

	handlerOption := inst.handlerOption.Load()
	if handlerOption.EnableFetch {
		fetcher, err := inst.fetcher.NewNativeFetch(r.Context())
		if err != nil {
			panic(fmt.Errorf("runtime panic: Failed to get native fetch: %w", err))
		}
		ctx.WithFetch(fetcher)
		defer inst.fetcher.DoneWith(fetcher)
	}

	if err := inst.resolver.NewPromiseFuncWithArg(
		middlewareHandler,
		ctx.NativeObject(),
		ctx.Resolve(),
		ctx.Reject(),
	); err != nil {
		ctx.Exception(err)
	}

	ctx.Wait()
}

func (inst *runtimeInstance) handleAsEvent(w http.ResponseWriter, r *http.Request, next http.Handler) {
	middlewareHandler := inst.middlewareHandler.Load().(goja.Value)

	evt := inst.eventPool.Get()
	defer inst.eventPool.Put(evt)

	evt.WithHttp(w, r, next)

	handlerOption := inst.handlerOption.Load()
	if handlerOption.EnableFetch {
		fetcher, err := inst.fetcher.NewNativeFetch(r.Context())
		if err != nil {
			panic(fmt.Errorf("runtime panic: Failed to get native fetch: %w", err))
		}
		evt.WithFetch(fetcher)
		defer inst.fetcher.DoneWith(fetcher)
	}

	if err := inst.resolver.NewPromiseFuncWithArg(
		middlewareHandler,
		evt.NativeObject(),
		evt.Resolve(),
		evt.Reject(),
	); err != nil {
		evt.Exception(err)
	}

	evt.Wait()
}

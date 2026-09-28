package heresy

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dop251/goja"
)

var (
	ErrRuntimeNotReady     = fmt.Errorf("middleware runtime is not ready")
	ErrNoMiddlewareHandler = fmt.Errorf("middleware script has no handler configured")
)

func (rt *Runtime) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt.shardRun(func(i int, instance *runtimeInstance) {
			w.Header().Set("X-Heresy-Shard", strconv.Itoa(i))

			if instance == nilInstance {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, ErrRuntimeNotReady)
				return
			}

			middlewareType := instance.middlewareType.Load().(handlerType)
			if middlewareType == handlerTypeUnset {
				defer instance.active.Done()
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

	ioCtx := inst.ioContextPool.GetWithLifetime(r.Context(), inst.context, inst.stopped)
	ioCtx.RegisterCleanup(inst.active.Done)
	defer inst.ioContextPool.Put(ioCtx)

	ctx := inst.contextPool.Get(ioCtx)
	if ctx == nil {
		http.Error(w, ErrRuntimeNotReady.Error(), http.StatusServiceUnavailable)
		return
	}

	ctx.WithHttp(w, r.WithContext(ioCtx.Context()), next)

	handlerOption := inst.handlerOption.Load()
	if handlerOption.EnableFetch {
		ctx.EnableFetch()
	}

	if err := inst.resolver.NewPromiseFuncWithArg(
		middlewareHandler,
		ctx.NativeObject(),
		ctx.Resolve(),
		ctx.Reject(),
	); err != nil {
		if inst.context.Err() != nil {
			<-inst.stopped
		} else {
			ctx.Exception(err)
		}
	}

	ctx.Wait()
}

func (inst *runtimeInstance) handleAsEvent(w http.ResponseWriter, r *http.Request, next http.Handler) {
	middlewareHandler := inst.middlewareHandler.Load().(goja.Value)

	ioCtx := inst.ioContextPool.GetWithLifetime(r.Context(), inst.context, inst.stopped)
	ioCtx.RegisterCleanup(inst.active.Done)
	defer inst.ioContextPool.Put(ioCtx)

	evt := inst.eventPool.Get(ioCtx)
	if evt == nil {
		http.Error(w, ErrRuntimeNotReady.Error(), http.StatusServiceUnavailable)
		return
	}

	evt.WithHttp(w, r.WithContext(ioCtx.Context()), next)

	handlerOption := inst.handlerOption.Load()
	if handlerOption.EnableFetch {
		evt.EnableFetch()
	}

	if err := inst.resolver.NewPromiseFuncWithArg(
		middlewareHandler,
		evt.NativeObject(),
		evt.Resolve(),
		evt.Reject(),
	); err != nil {
		if inst.context.Err() != nil {
			<-inst.stopped
		} else {
			evt.Exception(err)
		}
	}

	evt.Wait()
}

package heresy

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
)

var resolverPool = sync.Pool{
	New: func() any {
		return &requestResolver{
			done: make(chan struct{}, 1),
		}
	},
}

func getRequestResolver(w http.ResponseWriter, req *http.Request) *requestResolver {
	r := resolverPool.Get().(*requestResolver)
	r.respWriter, r.req = w, req
	return r
}

type requestResolver struct {
	respWriter http.ResponseWriter
	req        *http.Request
	done       chan struct{}
}

func (r *requestResolver) nativeResolveCallback(vm *goja.Runtime, scheduler *pond.WorkerPool) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		scheduler.Submit(func() {
			select {
			case <-r.req.Context().Done():
			default:
				v := fc.Argument(0)
				if goja.Undefined().Equals(v) {
					return
				}
				fmt.Fprint(r.respWriter, v)
			}
			r.done <- struct{}{}
		})
		return goja.Undefined()
	}
}

func (r *requestResolver) nativeRejectCallback(vm *goja.Runtime, scheduler *pond.WorkerPool) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		scheduler.Submit(func() {
			select {
			case <-r.req.Context().Done():
			default:
				v := fc.Argument(0)
				if goja.Undefined().Equals(v) {
					return
				}
				r.respWriter.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(r.respWriter, "Execution exception: %+v", v)
			}
			r.done <- struct{}{}
		})
		return goja.Undefined()
	}
}

func (r *requestResolver) exceptionCallback(err error, scheduler *pond.WorkerPool) {
	scheduler.Submit(func() {
		select {
		case <-r.req.Context().Done():
		default:
			r.respWriter.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(r.respWriter, "Unhandled exception: %+v", err)
		}
		r.done <- struct{}{}
	})
}

func (r *requestResolver) Wait() {
	<-r.done
	r.respWriter = nil
	r.req = nil
	resolverPool.Put(r)
}

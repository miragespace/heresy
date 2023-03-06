package heresy

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
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
	r.respWriter = w
	r.req = req
	return r
}

type requestResolver struct {
	respWriter http.ResponseWriter
	req        *http.Request
	done       chan struct{}
}

func (r *requestResolver) NativeResolve(vm *goja.Runtime) func(v goja.Value) {
	return func(v goja.Value) {
		go func() {
			select {
			case <-r.req.Context().Done():
			default:
				fmt.Fprint(r.respWriter, v)
				r.done <- struct{}{}
			}
		}()
	}
}

func (r *requestResolver) NativeReject(vm *goja.Runtime) func(v goja.Value) {
	return func(v goja.Value) {
		go func() {
			select {
			case <-r.req.Context().Done():
			default:
				r.respWriter.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(r.respWriter, "Execution exception: %+v", v)
				r.done <- struct{}{}
			}
		}()
	}
}

func (r *requestResolver) Exception(err error) {
	go func() {
		select {
		case <-r.req.Context().Done():
		default:
			r.respWriter.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(r.respWriter, "Unhandled exception: %+v", err)
			r.done <- struct{}{}
		}
	}()
}

func (r *requestResolver) Wait(eventLoop *eventloop.EventLoop) {
	select {
	case <-r.req.Context().Done():
	case <-r.done:
		r.respWriter = nil
		r.req = nil
		resolverPool.Put(r)
	}
}

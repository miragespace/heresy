package heresy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type fetchConfig struct {
	context   context.Context
	eventLoop *eventloop.EventLoop
	scheduler *pond.WorkerPool
	client    *http.Client
}

func (f *fetchConfig) runtimeWrapper(vm *goja.Runtime, req *http.Request, resolve, reject func(interface{})) {
	f.scheduler.Submit(func() {
		result, err := f.runtimeFetch(req)
		if err != nil {
			f.eventLoop.RunOnLoop(func(*goja.Runtime) {
				reject(vm.NewGoError(err))
			})
		} else {
			f.eventLoop.RunOnLoop(func(*goja.Runtime) {
				resolve(result)
			})
		}
	})
}

func (f *fetchConfig) runtimeFetch(req *http.Request) (any, error) {
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return string(body), nil
}

func (f *fetchConfig) nativeFetch(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	val := vm.ToValue(promise)

	req := fc.Argument(0)
	if goja.Undefined().Equals(req) {
		reject(vm.NewTypeError("missing argument in fetch"))
		return val
	}

	var (
		r   *http.Request
		err error
	)

	v := req.ExportType()
	switch v.Kind() {
	case reflect.String:
		r, err = http.NewRequestWithContext(f.context, http.MethodGet, req.String(), nil)
	default:
		err = fmt.Errorf("not implemented")
	}
	if err != nil {
		reject(vm.NewGoError(err))
	} else {
		f.runtimeWrapper(vm, r, resolve, reject)
	}

	return val
}

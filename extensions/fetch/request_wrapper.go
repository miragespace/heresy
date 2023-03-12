package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/dop251/goja"
)

type nativeFetchWrapper struct {
	cfg FetchConfig
	ctx context.Context

	_doFetch  goja.Value
	_unsetCtx goja.Value
}

var _ goja.DynamicObject = (*nativeFetchWrapper)(nil)

func (f *nativeFetchWrapper) WithContext(ctx context.Context) {
	f.ctx = ctx
}

func (f *nativeFetchWrapper) Reset() {
	f.ctx = nil
}

func (f *nativeFetchWrapper) Get(key string) goja.Value {
	switch key {
	case "doFetch":
		return f._doFetch
	case "unsetCtx":
		return f._unsetCtx
	default:
		return goja.Undefined()
	}
}

func (f *nativeFetchWrapper) Set(key string, val goja.Value) bool {
	return false
}

func (f *nativeFetchWrapper) Has(key string) bool {
	return false
}

func (f *nativeFetchWrapper) Delete(key string) bool {
	return false
}

func (f *nativeFetchWrapper) Keys() []string {
	return []string{}
}

func (f *nativeFetchWrapper) UnsetCtx() {
	f.ctx = nil
}

func (f *nativeFetchWrapper) DoFetch(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	promise, resolve, reject := vm.NewPromise()
	ret = vm.ToValue(promise)

	var (
		reqURL                    = fc.Argument(0)
		reqMethod                 = fc.Argument(1)
		reqHeaders                = fc.Argument(2)
		reqBody                   = fc.Argument(3)
		result                    = newResponseProxy(vm, f.cfg.Stream)
		bodyType                  = reqBody.ExportType()
		url        string         = reqURL.String()
		method     string         = reqMethod.String()
		headers    map[string]any = reqHeaders.Export().(map[string]any)
		useBody    io.Reader      = nil
		cleanup    func()         = func() {}
	)

	if goja.IsUndefined(reqBody) || goja.IsNull(reqBody) {
		// no body
	} else if bodyType.Kind() == reflect.String {
		useBody = bytes.NewBufferString(reqBody.String())
	} else {
		// possibly wrapped ReadableStream
		stream := reqBody.ToObject(vm)
		wrapper := stream.Get("wrapper")
		w, ok := AsNativeWrapper(wrapper)
		if !ok {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(vm.NewGoError(ErrUnsupportedReadableStream))
			})
			return
		}
		useBody = w.GetReader()
		cleanup = func() {
			f.cfg.Stream.Close(w)
		}
	}

	go func() {
		defer cleanup()

		req, err := http.NewRequestWithContext(f.ctx, method, url, useBody)
		if err != nil {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(err)
			})
			return
		}

		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%s", v))
		}
		req.Header.Set("user-agent", UserAgent)

		resp, err := f.cfg.Client.Do(req)
		if err != nil {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(err)
			})
		} else {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				result.WithResponse(vm, resp)
				resolve(result.nativeObj)
			})
		}
	}()

	return
}

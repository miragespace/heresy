package fetch

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
	pool "github.com/libp2p/go-buffer-pool"
)

type NativeFetchWrapper struct {
	cfg       FetchConfig
	ioContext *common.IOContext
}

func (f *NativeFetchWrapper) doFetch(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	promise, resolve, reject := vm.NewPromise()
	ret = vm.ToValue(promise)

	var (
		reqURL                    = fc.Argument(0)
		reqMethod                 = fc.Argument(1)
		reqHeaders                = fc.Argument(2)
		reqBody                   = fc.Argument(3)
		result                    = f.cfg.Stream.GetResponseProxy(f.ioContext)
		bodyType                  = reqBody.ExportType()
		url        string         = reqURL.String()
		method     string         = reqMethod.String()
		headers    map[string]any = reqHeaders.Export().(map[string]any)
		useBody    io.Reader      = nil
	)

	if goja.IsUndefined(reqBody) || goja.IsNull(reqBody) {
		// no body
		useBody = http.NoBody
	} else if bodyType.Kind() == reflect.String {
		strBuf := pool.NewBufferString(reqBody.String())
		f.ioContext.RegisterCleanup(strBuf.Reset)
		useBody = strBuf
	} else {
		// possibly wrapped ReadableStream
		reader, ok := stream.AssertReader(reqBody, vm)
		if !ok {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(vm.NewGoError(ErrUnsupportedReadableStream))
			})
			return
		}
		useBody = reader
	}

	go func() {
		err := f.ioContext.AcquireFetchToken()
		if err != nil {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(vm.NewGoError(err))
			})
			return
		}
		defer f.ioContext.ReleaseFetchToken()

		req, err := http.NewRequestWithContext(f.ioContext.Context(), method, url, useBody)
		if err != nil {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				reject(vm.NewGoError(err))
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
				reject(vm.NewGoError(err))
			})
		} else {
			f.cfg.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
				result.WithResponse(f.ioContext, vm, resp)
				resolve(result.NativeObject())
			})
		}
	}()

	return
}

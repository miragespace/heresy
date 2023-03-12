package fetch

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"go.miragespace.co/heresy/extensions/common"

	"github.com/dop251/goja"
	pool "github.com/libp2p/go-buffer-pool"
)

type NativeFetchWrapper struct {
	cfg       FetchConfig
	ioContext *common.IOContext
	respPool  *responseProxyPool
}

func (f *NativeFetchWrapper) WithIOContext(t *common.IOContext) {
	f.ioContext = t
}

func (f *NativeFetchWrapper) DoFetch(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	promise, resolve, reject := vm.NewPromise()
	ret = vm.ToValue(promise)

	var (
		reqURL                    = fc.Argument(0)
		reqMethod                 = fc.Argument(1)
		reqHeaders                = fc.Argument(2)
		reqBody                   = fc.Argument(3)
		result                    = f.respPool.Get()
		bodyType                  = reqBody.ExportType()
		url        string         = reqURL.String()
		method     string         = reqMethod.String()
		headers    map[string]any = reqHeaders.Export().(map[string]any)
		useBody    io.Reader      = nil
	)
	f.ioContext.RegisterCleanup(func() {
		f.respPool.Put(result)
	})

	if goja.IsUndefined(reqBody) || goja.IsNull(reqBody) {
		// no body
	} else if bodyType.Kind() == reflect.String {
		strBuf := pool.NewBufferString(reqBody.String())
		f.ioContext.RegisterCleanup(strBuf.Reset)
		useBody = strBuf
	} else {
		// possibly wrapped ReadableStream
		reader, ok := common.AssertReader(reqBody, vm)
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
				reject(err)
			})
			return
		}
		defer f.ioContext.ReleaseFetchToken()

		req, err := http.NewRequestWithContext(f.ioContext.Context(), method, url, useBody)
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
				result.WithResponse(f.ioContext, vm, resp)
				resolve(result.nativeObj)
			})
		}
	}()

	return
}

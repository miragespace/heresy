package fetch

import (
	"bytes"
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
	} else if buffer, ok := reqBody.Export().(goja.ArrayBuffer); ok {
		useBody = bytes.NewReader(bytes.Clone(buffer.Bytes()))
	} else if bodyType.Kind() == reflect.String {
		strBuf := pool.NewBufferString(reqBody.String())
		f.ioContext.RegisterCleanup(strBuf.Reset)
		useBody = strBuf
	} else {
		// possibly wrapped ReadableStream
		reader, ok := stream.AssertReader(reqBody, vm)
		if !ok {
			reject(vm.NewGoError(ErrUnsupportedReadableStream))
			return
		}
		useBody = reader
	}

	t := f.ioContext
	ctx := t.Context()
	t.StartIO()
	go func() {
		var resp *http.Response
		err := func() error {
			if err := t.AcquireFetchToken(); err != nil {
				return err
			}
			defer t.ReleaseFetchToken()
			req, err := http.NewRequestWithContext(ctx, method, url, useBody)
			if err != nil {
				return err
			}
			for k, v := range headers {
				req.Header.Set(k, fmt.Sprint(v))
			}
			if req.Header.Get("User-Agent") == "" {
				req.Header.Set("User-Agent", UserAgent)
			}
			resp, err = f.cfg.Client.Do(req)
			return err
		}()
		t.CompleteIO(f.cfg.Eventloop, func(vm *goja.Runtime) {
			if err != nil {
				reject(vm.NewGoError(err))
			} else {
				if err := result.WithResponse(t, vm, resp); err != nil {
					reject(vm.NewGoError(err))
				} else {
					resolve(result.NativeObject())
				}
			}
		}, func() {
			if resp != nil {
				resp.Body.Close()
			}
		})
	}()

	return
}

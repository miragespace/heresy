package fetch

import (
	"fmt"
	"net/http"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
)

type responseProxy struct {
	stream       *stream.StreamController
	resp         *http.Response
	vm           *goja.Runtime
	headersProxy *common.HeadersProxy
	nativeObj    *goja.Object
	nativeBody   goja.Value
}

var _ goja.DynamicObject = (*responseProxy)(nil)

func newResponseProxy(vm *goja.Runtime, controller *stream.StreamController) *responseProxy {
	r := &responseProxy{
		vm:           vm,
		stream:       controller,
		headersProxy: common.NewHeadersProxy(vm),
		nativeBody:   goja.Null(),
	}
	r.nativeObj = vm.NewDynamicObject(r)
	return r
}

func (r *responseProxy) WithResponse(vm *goja.Runtime, resp *http.Response) {
	var err error
	r.nativeBody, err = r.stream.NewReadableStreamVM(resp.Body, vm)
	if err != nil {
		panic(fmt.Errorf("runtime panic: Failed to convert httpResp.Body into native ReadableStream: %w", err))
	}
	r.headersProxy.UseHeader(resp.Header)
	r.resp = resp
}

func (r *responseProxy) Reset() {
	r.headersProxy.UnsetHeader()
	r.nativeBody = goja.Null()
	r.resp = nil
}

func (r *responseProxy) Get(key string) goja.Value {
	switch key {
	case "statusText":
		return r.vm.ToValue(r.resp.Status)
	case "statusCode":
		return r.vm.ToValue(r.resp.StatusCode)

	case "headers":
		return r.headersProxy.NativeObject()
	case "body":
		return r.nativeBody

	default:
		return goja.Undefined()
	}
}

func (r *responseProxy) Set(key string, val goja.Value) bool {
	return false
}

func (r *responseProxy) Has(key string) bool {
	return !goja.IsUndefined(r.Get(key))
}

func (r *responseProxy) Delete(key string) bool {
	return false
}

func (r *responseProxy) Keys() []string {
	return []string{"body", "header", "statusCode", "statusText"}
}

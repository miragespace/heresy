package fetch

import (
	"net/http"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
)

type responseProxy struct {
	stream       *stream.StreamController
	ioContext    *common.IOContext
	resp         *http.Response
	vm           *goja.Runtime
	headersProxy *common.HeadersProxy
	nativeObj    *goja.Object
	nativeBody   goja.Value
}

var _ goja.DynamicObject = (*responseProxy)(nil)

func newResponseProxy(vm *goja.Runtime, controller *stream.StreamController) *responseProxy {
	r := &responseProxy{
		vm:         vm,
		stream:     controller,
		nativeBody: goja.Null(),
	}
	r.nativeObj = vm.NewDynamicObject(r)
	return r
}

func (r *responseProxy) WithResponse(t *common.IOContext, vm *goja.Runtime, resp *http.Response) {
	readable := r.stream.NewReadableStreamVM(t, resp.Body, vm)
	r.nativeBody = readable.NativeStream()
	r.resp = resp
	r.ioContext = t
	t.RegisterCleanup(r.reset)
}

func (r *responseProxy) reset() {
	r.nativeBody = goja.Null()
	r.headersProxy = nil
	r.resp = nil
}

func (r *responseProxy) Get(key string) goja.Value {
	switch key {
	case "statusText":
		return r.vm.ToValue(r.resp.Status)
	case "statusCode":
		return r.vm.ToValue(r.resp.StatusCode)

	case "headers":
		// lazy initialization
		if r.headersProxy == nil {
			r.headersProxy = r.ioContext.GetHeadersProxy()
			r.headersProxy.UseHeader(r.resp.Header)
		}
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

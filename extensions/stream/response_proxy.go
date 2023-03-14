package stream

import (
	"net/http"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/shared"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
)

type ResponseProxy struct {
	stream                 *StreamController
	ioContext              *common.IOContext
	resp                   *http.Response
	vm                     *goja.Runtime
	headersProxy           *shared.HeadersProxy
	nativeResponseInstance *goja.Object
	nativeObj              *goja.Object
	nativeBody             goja.Value
}

var _ goja.DynamicObject = (*ResponseProxy)(nil)

func newResponseProxy(vm *goja.Runtime, controller *StreamController, symbols *polyfill.RuntimeSymbols) *ResponseProxy {
	r := &ResponseProxy{
		vm:                     vm,
		stream:                 controller,
		nativeBody:             goja.Null(),
		nativeResponseInstance: symbols.Response(),
	}
	r.nativeObj = vm.NewDynamicObject(r)
	r.nativeObj.SetPrototype(symbols.ResponsePrototype())
	return r
}

func (r *ResponseProxy) NativeObject() goja.Value {
	return r.nativeObj
}

func (r *ResponseProxy) WithResponse(t *common.IOContext, vm *goja.Runtime, resp *http.Response) {
	readable := r.stream.NewReadableStreamVM(t, resp.Body, vm)
	r.nativeBody = readable.NativeStream()
	r.resp = resp
	r.ioContext = t
	t.RegisterCleanup(r.reset)
}

func (r *ResponseProxy) reset() {
	r.ioContext = nil
	r.resp = nil
	r.headersProxy = nil
	r.nativeBody = goja.Null()
}

func (r *ResponseProxy) Get(key string) goja.Value {
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
		return r.nativeResponseInstance.Get(key)
	}
}

func (r *ResponseProxy) Set(key string, val goja.Value) bool {
	return false
}

func (r *ResponseProxy) Has(key string) bool {
	return !goja.IsUndefined(r.Get(key))
}

func (r *ResponseProxy) Delete(key string) bool {
	return false
}

func (r *ResponseProxy) Keys() []string {
	return []string{"body", "header", "statusCode", "statusText"}
}

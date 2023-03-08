package heresy

import (
	"fmt"
	"net"
	"net/http"

	"github.com/dop251/goja"
)

type contextRequest struct {
	*requestContext
	nativeReq      *goja.Object
	nativeGetValue goja.Value
}

var _ goja.DynamicObject = (*contextRequest)(nil)

func newContextRequest(ctx *requestContext) *contextRequest {
	req := &contextRequest{
		requestContext: ctx,
	}
	req.nativeReq = ctx.vm.NewDynamicObject(req)
	req.nativeGetValue = ctx.vm.ToValue(req.nativeGet)
	return req
}

func (req *contextRequest) Get(key string) goja.Value {
	httpReq := req.httpReq.Load().(*http.Request)

	switch key {
	case "ip":
		ip, _, _ := net.SplitHostPort(httpReq.RemoteAddr)
		return req.vm.ToValue(ip)
	case "method":
		return req.vm.ToValue(httpReq.Method)
	case "path":
		return req.vm.ToValue(httpReq.URL.Path)
	case "protocol":
		if httpReq.TLS == nil {
			return req.vm.ToValue("http")
		} else {
			return req.vm.ToValue("https")
		}
	case "secure":
		if httpReq.TLS == nil {
			return req.vm.ToValue(false)
		} else {
			return req.vm.ToValue(true)
		}

	case "get":
		return req.nativeGetValue
	case "res":
		return req.responseProxy.nativeRes

	default:
		return goja.Undefined()
	}
}

func (req *contextRequest) Set(_ string, _ goja.Value) bool {
	return false
}

func (req *contextRequest) Has(key string) bool {
	return !goja.IsUndefined(req.Get(key))
}

func (req *contextRequest) Delete(_ string) bool {
	return false
}

func (req *contextRequest) Keys() []string {
	return []string{"ip", "method", "path", "protocol", "secure", "get", "res"}
}

// implement Request.get(field) of Express.js
func (req *contextRequest) nativeGet(fc goja.FunctionCall) goja.Value {
	field := fc.Argument(0)
	if goja.IsUndefined(field) {
		panic(req.vm.NewTypeError("unexpected undefined to .get()"))
	}

	httpReq := req.httpReq.Load().(*http.Request)

	k := fmt.Sprintf("%s", field.Export())
	v := httpReq.Header.Get(k)

	if v != "" {
		return req.vm.ToValue(v)
	}

	return goja.Undefined()
}

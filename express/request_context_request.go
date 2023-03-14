package express

import (
	"net"

	"github.com/dop251/goja"
)

type contextRequest struct {
	*RequestContext
	nativeReq           *goja.Object
	nativeGet           goja.Value
	nativeReqProperties map[string]goja.Value
}

var _ goja.DynamicObject = (*contextRequest)(nil)

var requestProperties = []string{"ip", "method", "path", "protocol", "res", "secure"}

func newContextRequest(ctx *RequestContext) *contextRequest {
	req := &contextRequest{
		RequestContext:      ctx,
		nativeReqProperties: map[string]goja.Value{},
	}
	req.nativeReq = ctx.vm.NewDynamicObject(req)
	return req
}

func (req *contextRequest) reset() {
	for k := range req.nativeReqProperties {
		delete(req.nativeReqProperties, k)
	}
}

func (req *contextRequest) initReqProperty(key string) {
	var (
		r   = req.httpReq
		val goja.Value
	)

	switch key {
	case "ip":
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		val = req.vm.ToValue(ip)
	case "method":
		val = req.vm.ToValue(r.Method)
	case "path":
		val = req.vm.ToValue(r.URL.Path)
	case "protocol":
		if r.TLS == nil {
			val = req.vm.ToValue("http")
		} else {
			val = req.vm.ToValue("https")
		}
	case "secure":
		if r.TLS == nil {
			val = req.vm.ToValue(false)
		} else {
			val = req.vm.ToValue(true)
		}
	}

	req.nativeReqProperties[key] = val
}

func (req *contextRequest) Get(key string) goja.Value {
	if req.Has(key) {
		if req.nativeReqProperties[key] == nil {
			req.initReqProperty(key)
		}
		return req.nativeReqProperties[key]
	}

	switch key {
	case "get":
		if req.nativeGet == nil {
			req.nativeGet = req.vm.ToValue(req.get)
		}
		return req.nativeGet
	case "res":
		if req.responseProxy == nil {
			req.responseProxy = newContextResponse(req.RequestContext)
		}
		return req.responseProxy.nativeRes

	default:
		return goja.Undefined()
	}
}

func (req *contextRequest) Set(_ string, _ goja.Value) bool {
	return false
}

func (req *contextRequest) Has(key string) bool {
	for _, k := range requestProperties {
		if k == key {
			return true
		}
	}
	return false
}

func (req *contextRequest) Delete(_ string) bool {
	return false
}

func (req *contextRequest) Keys() []string {
	return requestProperties
}

// implement Request.get(field) of Express.js
func (req *contextRequest) get(fc goja.FunctionCall) goja.Value {
	field := fc.Argument(0)
	if goja.IsUndefined(field) {
		panic(req.vm.NewTypeError("unexpected undefined to .get()"))
	}

	var v string
	if s, ok := field.Export().(string); ok {
		v = req.httpReq.Header.Get(s)
	}

	if v != "" {
		return req.vm.ToValue(v)
	}

	return goja.Undefined()
}

package event

import (
	"fmt"
	"net/http"

	"go.miragespace.co/heresy/extensions/common/shared"

	"github.com/dop251/goja"
)

type fetchEventRequest struct {
	*FetchEvent
	headersProxy          *shared.HeadersProxy
	nativeBody            goja.Value
	nativeReq             *goja.Object
	nativeRequestInstance *goja.Object
	nativeProperties      map[string]goja.Value
	bodyConsumed          bool
}

var _ goja.DynamicObject = (*fetchEventRequest)(nil)

var requestProperties = []string{"body", "bodyUsed", "headers", "method", "url"}

func newFetchEventRequest(evt *FetchEvent) *fetchEventRequest {
	req := &fetchEventRequest{
		FetchEvent:            evt,
		bodyConsumed:          false,
		nativeBody:            goja.Null(),
		nativeRequestInstance: evt.deps.Symbols.Request(),
		nativeProperties:      map[string]goja.Value{},
	}

	req.nativeReq = evt.vm.NewDynamicObject(req)
	req.nativeReq.SetPrototype(evt.deps.Symbols.RequestPrototype())

	return req
}

func (req *fetchEventRequest) initializeBody() {
	switch req.httpReq.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		readable := req.deps.Stream.NewReadableStreamVM(req.ioContext, req.httpReq.Body, req.vm)
		req.nativeBody = readable.NativeStream()
	}
}

func (req *fetchEventRequest) reset() {
	req.bodyConsumed = false
	req.nativeBody = goja.Null()
	req.headersProxy = nil
	for k := range req.nativeProperties {
		delete(req.nativeProperties, k)
	}
}

func makeUrl(r *http.Request) string {
	scheme := "http"
	if r.TLS == nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())
}

func (req *fetchEventRequest) Get(key string) goja.Value {
	switch key {
	case "url":
		if req.nativeProperties[key] == nil {
			req.nativeProperties[key] = req.vm.ToValue(makeUrl(req.httpReq))
		}
		return req.nativeProperties[key]
	case "method":
		if req.nativeProperties[key] == nil {
			req.nativeProperties[key] = req.vm.ToValue(req.httpReq.Method)
		}
		return req.nativeProperties[key]

	// NOTE: since this is a fake Request object, any access to the properties in Body class
	// needs to be handled by us as these fields won't be set by the constructor.
	case "bodyUsed", "_consumed":
		return req.vm.ToValue(req.bodyConsumed)
	case "body", "bodyInit", "_bodyReadableStream":
		if goja.IsNull(req.nativeBody) {
			req.initializeBody()
		}
		return req.nativeBody

	case "headers":
		if req.headersProxy == nil {
			req.headersProxy = req.ioContext.GetHeadersProxy()
			req.headersProxy.UseHeader(req.httpReq.Header)
		}
		return req.headersProxy.NativeObject()

	default:
		return req.nativeRequestInstance.Get(key)
	}
}

func (req *fetchEventRequest) Set(key string, val goja.Value) bool {
	switch key {
	case "_consumed":
		req.bodyConsumed = val.ToBoolean()
		return true
	default:
		return false
	}
}

func (req *fetchEventRequest) Has(key string) bool {
	for _, k := range requestProperties {
		if k == key {
			return true
		}
	}
	return false
}

func (req *fetchEventRequest) Delete(key string) bool {
	return false
}

func (req *fetchEventRequest) Keys() []string {
	return requestProperties
}

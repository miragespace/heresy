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
	bodyConsumed          bool
}

var _ goja.DynamicObject = (*fetchEventRequest)(nil)

func newFetchEventRequest(evt *FetchEvent) *fetchEventRequest {
	req := &fetchEventRequest{
		FetchEvent:            evt,
		bodyConsumed:          false,
		nativeBody:            goja.Null(),
		nativeRequestInstance: evt.deps.Symbols.Request(),
	}

	req.nativeReq = evt.vm.NewDynamicObject(req)
	req.nativeReq.SetPrototype(req.nativeRequestInstance.Prototype())

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
}

func (req *fetchEventRequest) Get(key string) goja.Value {
	switch key {
	case "url":
		scheme := "http"
		if req.httpReq.TLS == nil {
			scheme = "https"
		}
		return req.vm.ToValue(fmt.Sprintf("%s://%s%s", scheme, req.httpReq.Host, req.httpReq.URL.RequestURI()))
	case "method":
		return req.vm.ToValue(req.httpReq.Method)

	// NOTE: since this is a fake Request object, any access to the properties in Body class
	// needs to be handled by us as these fields won't be set by the constructor.
	case "bodyUsed", "_consumed":
		return req.vm.ToValue(req.bodyConsumed)
	case "body", "bodyInit", "_bodyReadableStream":
		// lazy initialization
		if goja.IsNull(req.nativeBody) {
			req.initializeBody()
		}
		return req.nativeBody

	case "headers":
		// lazy initialization
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
	return !goja.IsUndefined(req.Get(key))
}

func (req *fetchEventRequest) Delete(key string) bool {
	return false
}

func (req *fetchEventRequest) Keys() []string {
	return []string{"body", "bodyUsed", "headers", "method", "url"}
}

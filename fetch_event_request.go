package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
	"go.miragespace.co/heresy/extensions/common"
)

type fetchEventRequest struct {
	*fetchEvent
	headersProxy          *common.HeadersProxy
	nativeBody            goja.Value
	nativeReq             *goja.Object
	nativeRequestInstance *goja.Object
	bodyConsumed          bool
}

var _ goja.DynamicObject = (*fetchEventRequest)(nil)

func newFetchEventRequest(evt *fetchEvent) *fetchEventRequest {
	req := &fetchEventRequest{
		fetchEvent:   evt,
		nativeBody:   goja.Null(),
		bodyConsumed: false,
	}
	req.headersProxy = common.NewHeadersProxy(evt.vm)

	requestClass := req.vm.Get("Request")
	requestConstructor, ok := goja.AssertConstructor(requestClass)
	if !ok {
		panic("runtime panic: Request is not a constructor, please check if polyfill is enabled")
	}

	var err error
	req.nativeRequestInstance, err = requestConstructor(nil)
	if err != nil {
		panic(fmt.Errorf("runtime panic: (new Request) constructor call returned an error: %w", err))
	}

	req.nativeReq = evt.vm.NewDynamicObject(req)
	req.nativeReq.SetPrototype(req.nativeRequestInstance.Prototype())

	return req
}

func (req *fetchEventRequest) initialize() {
	switch req.httpReq.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		nativeBody, err := req.stream.NewReadableStream(req.httpReq.Body)
		if err != nil {
			panic(fmt.Errorf("runtime panic: Failed to convert httpReq.Body into native ReadableStream: %w", err))
		}
		req.nativeBody = nativeBody
	}
	req.headersProxy.UseHeader(req.httpReq.Header)
}

func (req *fetchEventRequest) Reset() {
	req.headersProxy.UnsetHeader()
	req.nativeBody = goja.Null()
	req.bodyConsumed = false
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
		return req.nativeBody

	case "headers":
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

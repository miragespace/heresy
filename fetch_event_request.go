package heresy

import (
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

type fetchEventRequest struct {
	*fetchEvent
	headersProxy          *fetchEventRequestHeaders
	nativeBody            goja.Value
	nativeReq             *goja.Object
	nativeRequestInstance *goja.Object
	bodyConsumed          bool
}

// TODO: need to make Headers and Request as DynamicObjects
func newFetchEventRequest(evt *fetchEvent) *fetchEventRequest {
	req := &fetchEventRequest{
		fetchEvent:   evt,
		nativeBody:   goja.Null(),
		bodyConsumed: false,
	}
	req.headersProxy = newFetchEventRequestHeaders(evt)

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
	req.nativeReq.SetPrototype(req.nativeRequestInstance)

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

}

func (req *fetchEventRequest) reset() {
	req.nativeBody = goja.Null()
	req.bodyConsumed = false
}

func (req *fetchEventRequest) Get(key string) goja.Value {
	switch key {
	case "url":
		if req.httpReq.URL.RawQuery != "" {
			return req.vm.ToValue(req.httpReq.URL.Path + "?" + req.httpReq.URL.RawQuery)
		}
		return req.vm.ToValue(req.httpReq.URL.Path)
	case "method":
		return req.vm.ToValue(req.httpReq.Method)
	case "_consumed":
		return req.vm.ToValue(req.bodyConsumed)

	case "bodyInit":
		fallthrough
	case "_bodyReadableStream":
		return req.nativeBody

	case "headers":
		return req.headersProxy.nativeHeaders

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
	return []string{"bodyInit", "url", "method", "headers"}
}

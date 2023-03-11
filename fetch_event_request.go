package heresy

import (
	"strings"

	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
)

type fetchEventRequest struct {
	*fetchEvent
	nativeReq *goja.Object
}

type requestInit struct {
	Body    goja.Value        `json:"body,omitempty"`
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`
}

// TODO: need to make Headers and Request as DynamicObjects
func newFetchEventRequest(evt *fetchEvent, controller *stream.StreamController) *fetchEventRequest {
	req := &fetchEventRequest{
		fetchEvent: evt,
	}

	var (
		runtimeRequestConstructor goja.Constructor
		nativeBody                goja.Value
		ok                        bool
		err                       error
	)
	constructor := req.vm.Get("Request")
	runtimeRequestConstructor, ok = goja.AssertConstructor(constructor)
	if !ok {
		panic("runtime panic: Request is not a constructor, please check if polyfill is enabled")
	}

	init := requestInit{
		Method:  req.httpReq.Method,
		Headers: map[string]string{},
	}

	switch init.Method {
	case "GET":
	case "HEAD":
	default:
		nativeBody, err = controller.NewReadableStreamVM(evt.httpReq.Body, req.vm)
		if err != nil {
			panic("runtime panic: Failed to convert httpReq.Body into native ReadableStream")
		}
		init.Body = nativeBody
	}

	for key, vals := range evt.httpReq.Header {
		init.Headers[key] = strings.Join(vals, ", ")
	}

	u := evt.httpReq.URL
	var path string
	if u.RawQuery != "" {
		path = strings.Join([]string{u.Path, u.RawQuery}, "?")
	} else {
		path = strings.Join([]string{u.Path}, "?")
	}

	req.nativeReq, err = runtimeRequestConstructor(
		nil,
		req.vm.ToValue(path),
		req.vm.ToValue(&init),
	)
	if err != nil {
		panic("runtime panic: (new Request) constructor call returned an error")
	}

	return req
}

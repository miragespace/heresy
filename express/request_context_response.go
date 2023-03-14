package express

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/dop251/goja"
)

type contextResponse struct {
	*RequestContext
	nativeRes       *goja.Object
	nativeRespFuncs map[string]goja.Value
	statusCode      int
}

var _ goja.DynamicObject = (*contextResponse)(nil)

var responseProperties = []string{"headersSent"}

func newContextResponse(ctx *RequestContext) *contextResponse {
	res := &contextResponse{
		RequestContext:  ctx,
		nativeRespFuncs: map[string]goja.Value{},
		statusCode:      http.StatusNoContent,
	}
	res.nativeRes = ctx.vm.NewDynamicObject(res)
	return res
}

func (res *contextResponse) initFunction(key string) {
	var val goja.Value
	switch key {
	case "status":
		val = res.vm.ToValue(res.status)
	case "send":
		val = res.vm.ToValue(res.send)
	case "json":
		val = res.vm.ToValue(res.json)
	case "get":
		val = res.vm.ToValue(res.get)
	case "end":
		val = res.vm.ToValue(res.end)
	case "set":
		fallthrough
	case "header":
		val = res.vm.ToValue(res.set)
	}
	if val != nil {
		res.nativeRespFuncs[key] = val
	}
}

func (res *contextResponse) Get(key string) goja.Value {
	if res.Has(key) {
		switch key {
		case "headersSent":
			return res.vm.ToValue(res.responseSent)
		}
	}

	if res.nativeRespFuncs[key] == nil {
		res.initFunction(key)
	}
	if res.nativeRespFuncs[key] != nil {
		return res.nativeRespFuncs[key]
	}

	return goja.Undefined()
}

func (res *contextResponse) Set(_ string, _ goja.Value) bool {
	return false
}

func (res *contextResponse) Has(key string) bool {
	for _, k := range responseProperties {
		if k == key {
			return true
		}
	}
	return false
}

func (res *contextResponse) Delete(_ string) bool {
	return false
}

func (res *contextResponse) Keys() []string {
	return responseProperties
}

func (res *contextResponse) reset() {
	res.statusCode = http.StatusNoContent
	res.statusSet = false
}

// implement Response.get(field) of Express.js
func (res *contextResponse) get(fc goja.FunctionCall) goja.Value {
	field := fc.Argument(0)
	if goja.IsUndefined(field) {
		panic(res.vm.NewTypeError("unexpected undefined to .get()"))
	}

	w := res.httpResp

	var v string
	if s, ok := field.Export().(string); ok {
		v = w.Header().Get(s)
	}

	if v != "" {
		return res.vm.ToValue(v)
	}

	return goja.Undefined()
}

// implement Response.set(field [, value]) of Express.js (chainable)
func (res *contextResponse) set(fc goja.FunctionCall) goja.Value {
	field := fc.Argument(0)
	val := fc.Argument(1)

	if goja.IsUndefined(field) {
		panic(res.vm.NewTypeError("invalid undefined argument"))
	}

	if len(fc.Arguments) == 2 {
		k := fmt.Sprintf("%s", field.Export())
		v := fmt.Sprintf("%v", val.Export())

		if strings.ToLower(k) == "content-type" {
			if val.ExportType().Kind() == reflect.Slice {
				panic(res.vm.NewTypeError("Content-Type cannot be set to an Array"))
			}
			// TODO: add charset based on mime like express
		}

		// TODO: handle slice of values
		w := res.httpResp
		header := w.Header()
		if header.Get(k) == "" {
			header.Set(k, v)
		} else {
			header.Add(k, v)
		}
	} else {
		var m map[string]interface{}
		if err := res.vm.ExportTo(field, &m); err != nil {
			panic(res.vm.NewGoError(err))
		}
		for k, v := range m {
			res.set(goja.FunctionCall{
				This: fc.This,
				Arguments: []goja.Value{
					res.vm.ToValue(k),
					res.vm.ToValue(v),
				},
			})
		}
	}
	return res.nativeRes
}

// implement Response.status(code) of Express.js (chainable)
func (res *contextResponse) status(fc goja.FunctionCall) goja.Value {
	if res.responseSent {
		panic(res.vm.NewTypeError("response already sent"))
	}

	nativeCode := fc.Argument(0)
	if goja.IsUndefined(nativeCode) {
		panic(res.vm.NewTypeError("invalid parameter to .status()"))
	}

	var code int
	if err := res.vm.ExportTo(nativeCode, &code); err != nil {
		panic(res.vm.NewGoError(err))
	}
	if http.StatusText(code) == "" {
		panic(res.vm.NewTypeError("invalid http status code to .status()"))
	}

	res.statusCode = code
	res.statusSet = true

	return res.nativeRes
}

// implement Response.json([body]) of Express.js
func (res *contextResponse) json(fc goja.FunctionCall) goja.Value {
	if res.responseSent {
		panic(res.vm.NewTypeError("response already sent"))
	}

	body := fc.Argument(0)

	var (
		obj     *goja.Object
		content []byte
	)

	w := res.httpResp
	header := w.Header()
	header.Set("content-type", "application/json")

	if goja.IsUndefined(body) {
		content = append(content, "{}"...)
	} else if goja.IsNull(body) {
		content = []byte(string("null"))
	} else {
		obj = body.ToObject(res.vm)
		content, _ = obj.MarshalJSON()
	}

	res.sendHeaders()
	w.Write(content)

	return goja.Undefined()
}

// implement Response.end([body]) of Express.js
func (res *contextResponse) send(fc goja.FunctionCall) goja.Value {
	if res.responseSent {
		panic(res.vm.NewTypeError("response already sent"))
	}

	body := fc.Argument(0)
	if goja.IsUndefined(body) {
		return goja.Undefined()
	}

	var (
		content []byte
	)

	w := res.httpResp
	header := w.Header()

	if goja.IsNull(body) {
		return res.json(fc)
	} else {
		switch body.ExportType().Kind() {
		case reflect.String:
			if header.Get("content-type") == "" {
				header.Set("content-type", "text/html")
			}
			res.vm.ExportTo(body, &content)
		case reflect.Map, reflect.Slice, reflect.Bool:
			return res.json(fc)
		default:
			fmt.Printf("%+v\n", body.ExportType())
			fmt.Printf("%+v\n", body.ExportType().Kind() == reflect.Map)
			fmt.Printf("%+v\n", body.Export())
		}
	}

	res.sendHeaders()
	w.Write(content)

	return goja.Undefined()
}

// implement Response.end([data] [, encoding]) of Express.js
func (res *contextResponse) end(fc goja.FunctionCall) goja.Value {
	if res.responseSent {
		panic(res.vm.NewTypeError("response already sent"))
	}

	w := res.httpResp
	res.sendHeaders()

	body := fc.Argument(0)
	if goja.IsUndefined(body) {
		return goja.Undefined()
	}

	switch body.ExportType().Kind() {
	case reflect.String:
		var content []byte
		res.vm.ExportTo(body, &content)
		w.Write(content)
	}

	return goja.Undefined()
}

func (res *contextResponse) sendHeaders() {
	w := res.httpResp
	if res.statusSet {
		w.WriteHeader(res.statusCode)
	} else {
		w.WriteHeader(http.StatusOK)
		res.statusSet = true
	}
	res.responseSent = true
}

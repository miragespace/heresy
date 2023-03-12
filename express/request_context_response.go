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
	nativeRes  *goja.Object
	statusCode int
}

var _ goja.DynamicObject = (*contextResponse)(nil)

func newContextResponse(ctx *RequestContext) *contextResponse {
	res := &contextResponse{
		RequestContext: ctx,
	}
	res.reset()
	res.nativeRes = ctx.vm.NewDynamicObject(res)
	return res
}

func (res *contextResponse) Get(key string) goja.Value {
	switch key {
	case "status":
		return res.vm.ToValue(res.nativeStatus)
	case "send":
		return res.vm.ToValue(res.nativeSend)
	case "json":
		return res.vm.ToValue(res.nativeJson)
	case "get":
		return res.vm.ToValue(res.nativeGet)
	case "end":
		return res.vm.ToValue(res.nativeEnd)
	case "set":
		fallthrough
	case "header":
		return res.vm.ToValue(res.nativeSet)

	case "headersSent":
		return res.vm.ToValue(res.responseSent)

	default:
		return goja.Undefined()
	}
}

func (res *contextResponse) Set(_ string, _ goja.Value) bool {
	return false
}

func (res *contextResponse) Has(key string) bool {
	return !goja.IsUndefined(res.Get(key))
}

func (res *contextResponse) Delete(_ string) bool {
	return false
}

func (res *contextResponse) Keys() []string {
	return []string{"headersSent"}
}

func (res *contextResponse) reset() {
	res.statusCode = http.StatusNoContent
	res.statusSet = false
}

// implement Response.get(field) of Express.js
func (res *contextResponse) nativeGet(fc goja.FunctionCall) goja.Value {
	field := fc.Argument(0)
	if goja.IsUndefined(field) {
		panic(res.vm.NewTypeError("unexpected undefined to .get()"))
	}

	w := res.httpResp

	k := fmt.Sprintf("%s", field.Export())
	v := w.Header().Get(k)

	if v != "" {
		return res.vm.ToValue(v)
	}

	return goja.Undefined()
}

// implement Response.set(field [, value]) of Express.js (chainable)
func (res *contextResponse) nativeSet(fc goja.FunctionCall) goja.Value {
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
			res.nativeSet(goja.FunctionCall{
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
func (res *contextResponse) nativeStatus(fc goja.FunctionCall) goja.Value {
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
func (res *contextResponse) nativeJson(fc goja.FunctionCall) goja.Value {
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
func (res *contextResponse) nativeSend(fc goja.FunctionCall) goja.Value {
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
		return res.nativeJson(fc)
	} else {
		switch body.ExportType().Kind() {
		case reflect.String:
			if header.Get("content-type") == "" {
				header.Set("content-type", "text/html")
			}
			res.vm.ExportTo(body, &content)
		case reflect.Map, reflect.Slice, reflect.Bool:
			return res.nativeJson(fc)
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
func (res *contextResponse) nativeEnd(fc goja.FunctionCall) goja.Value {
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

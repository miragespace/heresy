package polyfill

import (
	"embed"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const (
	RuntimeFetchEventInstanceSymbol = "__runtimeFetchEventInstance"
	RuntimeRequestInstanceSymbol    = "__runtimeRequestInstance"
)

//go:embed node_modules/*
var PolyfillFS embed.FS

//go:embed polyfill.js
var polyfillScript string

var polyfillProg = goja.MustCompile("polyfill", polyfillScript, true)

type RuntimeSymbols struct {
	nativeRequestInstance    *goja.Object
	nativeFetchEventInstance *goja.Object
	nativeHeadersConstructor goja.Constructor
}

func (r *RuntimeSymbols) Request() *goja.Object {
	return r.nativeRequestInstance
}

func (r *RuntimeSymbols) FetchEvent() *goja.Object {
	return r.nativeFetchEventInstance
}

func (r *RuntimeSymbols) Headers() goja.Constructor {
	return r.nativeHeadersConstructor
}

func PolyfillRuntime(eventLoop *eventloop.EventLoop) (s *RuntimeSymbols, err error) {
	setup := make(chan error, 1)
	eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(polyfillProg)
		if err != nil {
			setup <- err
			return
		}

		reqeustInstance := vm.Get(RuntimeRequestInstanceSymbol)
		if goja.IsUndefined(reqeustInstance) {
			setup <- fmt.Errorf("polyfill symbols not found, please check if polyfill is configured correctly")
			return
		}
		nativeRequestInstance := reqeustInstance.ToObject(vm)

		fetchEventInstance := vm.Get(RuntimeFetchEventInstanceSymbol)
		if goja.IsUndefined(fetchEventInstance) {
			setup <- fmt.Errorf("polyfill symbols not found, please check if polyfill is configured correctly")
			return
		}
		nativeEvtInstance := fetchEventInstance.ToObject(vm)

		headesrClass := vm.Get("Headers")
		headersConstructor, ok := goja.AssertConstructor(headesrClass)
		if !ok {
			setup <- fmt.Errorf("runtime panic: Headers is not a constructor, please check if polyfill is configured correctly")
			return
		}

		s = &RuntimeSymbols{
			nativeRequestInstance:    nativeRequestInstance,
			nativeFetchEventInstance: nativeEvtInstance,
			nativeHeadersConstructor: headersConstructor,
		}

		setup <- nil
	})

	err = <-setup
	return
}

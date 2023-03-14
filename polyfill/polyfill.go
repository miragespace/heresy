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
	RuntimeResponseInstanceSymbol   = "__runtimeResponseInstance"
)

//go:embed node_modules/*
var PolyfillFS embed.FS

//go:embed polyfill.js
var polyfillScript string

var polyfillProg = goja.MustCompile("polyfill", polyfillScript, true)

type RuntimeSymbols struct {
	nativeRequestInstancePrototype    *goja.Object
	nativeResponseInstancePrototype   *goja.Object
	nativeFetchEventInstancePrototype *goja.Object
	nativeRequestInstance             *goja.Object
	nativeResponseInstance            *goja.Object
	nativeFetchEventInstance          *goja.Object
	nativeHeadersConstructor          goja.Constructor
}

func (r *RuntimeSymbols) Request() *goja.Object {
	return r.nativeRequestInstance
}

func (r *RuntimeSymbols) RequestPrototype() *goja.Object {
	return r.nativeRequestInstancePrototype
}

func (r *RuntimeSymbols) Response() *goja.Object {
	return r.nativeResponseInstance
}

func (r *RuntimeSymbols) ResponsePrototype() *goja.Object {
	return r.nativeResponseInstancePrototype
}

func (r *RuntimeSymbols) FetchEvent() *goja.Object {
	return r.nativeFetchEventInstance
}

func (r *RuntimeSymbols) FetchEventPrototype() *goja.Object {
	return r.nativeFetchEventInstancePrototype
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

		s = &RuntimeSymbols{}

		for _, assignment := range []struct {
			target **goja.Object
			proto  **goja.Object
			name   string
		}{
			{
				target: &s.nativeRequestInstance,
				proto:  &s.nativeRequestInstancePrototype,
				name:   RuntimeRequestInstanceSymbol,
			},
			{
				target: &s.nativeResponseInstance,
				proto:  &s.nativeResponseInstancePrototype,
				name:   RuntimeResponseInstanceSymbol,
			},
			{
				target: &s.nativeFetchEventInstance,
				proto:  &s.nativeFetchEventInstancePrototype,
				name:   RuntimeFetchEventInstanceSymbol,
			},
		} {
			instance := vm.Get(assignment.name)
			if goja.IsUndefined(instance) {
				setup <- fmt.Errorf("polyfill symbols not found, please check if polyfill is configured correctly")
				return
			}
			obj := instance.ToObject(vm)
			*assignment.target = obj
			*assignment.proto = obj.Prototype()
		}

		headesrClass := vm.Get("Headers")
		headersConstructor, ok := goja.AssertConstructor(headesrClass)
		if !ok {
			setup <- fmt.Errorf("runtime panic: Headers is not a constructor, please check if polyfill is configured correctly")
			return
		}
		s.nativeHeadersConstructor = headersConstructor

		setup <- nil
	})

	err = <-setup
	return
}

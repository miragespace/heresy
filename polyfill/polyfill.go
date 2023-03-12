package polyfill

import (
	"embed"

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

func PolyfillRuntime(eventLoop *eventloop.EventLoop) error {
	setup := make(chan error, 1)
	eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(polyfillProg)
		if err != nil {
			setup <- err
			return
		}
		setup <- nil
	})

	return <-setup
}

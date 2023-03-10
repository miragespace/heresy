package modules

import (
	"embed"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

//go:embed node_modules/*
var ModulesFS embed.FS

//go:embed modules.js
var modulesExporterScript string

var modulesExporterProg = goja.MustCompile("modulesExporter", modulesExporterScript, true)

func InjectModules(eventLoop *eventloop.EventLoop) error {
	setup := make(chan error, 1)
	eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(modulesExporterProg)
		if err != nil {
			setup <- err
			return
		}
		setup <- nil
	})

	return <-setup
}

package heresy

import (
	"fmt"
	"sync/atomic"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"go.uber.org/zap"
)

type Runtime struct {
	httpHandler     atomic.Value // goja.Callable
	runtimeResolver atomic.Value // goja.Callable
	registry        *require.Registry
	eventLoop       *eventloop.EventLoop
}

func NewRuntime(logger *zap.Logger, scriptName string, script string) (*Runtime, error) {
	prog, err := goja.Compile(scriptName, script, true)
	if err != nil {
		return nil, fmt.Errorf("error compiling script: %w", err)
	}

	useZapLogger := logger != nil

	registry := require.NewRegistry()
	eventLoop := eventloop.NewEventLoop(
		eventloop.EnableConsole(!useZapLogger),
		eventloop.WithRegistry(registry),
	)

	rt := &Runtime{
		registry:  registry,
		eventLoop: eventLoop,
	}

	eventLoop.Start()

	if useZapLogger {
		runtimeLogger := newZapPrinter(scriptName, logger)

		loggerModule := console.RequireWithPrinter(runtimeLogger)
		require.RegisterNativeModule(moduleName, loggerModule)

		eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			vm.Set("console", require.Require(vm, moduleName))
		})
	}

	setup := make(chan error, 1)
	rt.setupRuntime(prog, setup)
	err = <-setup
	if err != nil {
		return nil, err
	}

	return rt, nil
}

func (rt *Runtime) Stop() {
	rt.eventLoop.Stop()
}

func (rt *Runtime) setupRuntime(prog *goja.Program, setup chan error) {
	rt.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		var err error
		_, err = vm.RunProgram(runtimeResolverProg)
		if err != nil {
			setup <- fmt.Errorf("internal error: failed to setting up runtime resolver: %w", err)
			return
		}

		runtimeResolver := vm.Get("__runtimeResolver")
		resolver, ok := goja.AssertFunction(runtimeResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: __runtimeResolver is not a function")
			return
		}
		rt.runtimeResolver.Store(resolver)

		vm.Set("onRequest", func(fn goja.Value) {
			if _, ok := goja.AssertFunction(fn); ok {
				rt.httpHandler.Store(fn)
			}
		})
		WithFetch(rt.eventLoop, vm, nil)

		_, err = vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up request script: %w", err)
			return
		}

		setup <- nil
	})
}

package heresy

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"go.uber.org/zap"
)

type Runtime struct {
	instance  atomic.Pointer[runtimeInstance]
	registry  *require.Registry
	scheduler *pond.WorkerPool
}

type runtimeInstance struct {
	closeOnce       sync.Once
	running         atomic.Bool
	httpHandler     atomic.Value // goja.Value
	runtimeResolver goja.Callable
	eventLoop       *eventloop.EventLoop
}

func NewRuntime(logger *zap.Logger, scriptName string, script string) (*Runtime, error) {
	prog, err := goja.Compile(scriptName, script, true)
	if err != nil {
		return nil, fmt.Errorf("error compiling script: %w", err)
	}

	useZapLogger := logger != nil

	rt := &Runtime{
		registry:  require.NewRegistry(),
		scheduler: pond.New(100, 1000, pond.MinWorkers(4)),
	}

	eventLoop := eventloop.NewEventLoop(
		eventloop.EnableConsole(!useZapLogger),
		eventloop.WithRegistry(rt.registry),
	)
	eventLoop.Start()

	if useZapLogger {
		runtimeLogger := newZapPrinter(scriptName, logger)

		loggerModule := console.RequireWithPrinter(runtimeLogger)
		require.RegisterNativeModule(moduleName, loggerModule)

		eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			vm.Set("console", require.Require(vm, moduleName))
		})
	}

	instance := &runtimeInstance{
		eventLoop: eventLoop,
	}

	err = <-rt.setupRuntime(prog, instance)
	if err != nil {
		rt.scheduler.Stop()
		eventLoop.StopNoWait()
		return nil, err
	}

	rt.instance.Store(instance)
	return rt, nil
}

func (rt *Runtime) Stop() {
	inst := rt.instance.Load()
	inst.closeOnce.Do(func() {
		inst.running.Store(false)
		inst.eventLoop.StopNoWait()
	})
	rt.scheduler.Stop()
}

func (rt *Runtime) setupRuntime(prog *goja.Program, inst *runtimeInstance) chan error {
	setup := make(chan error, 1)
	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
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
		inst.runtimeResolver = resolver

		vm.Set("registerRequestHandler", func(fn goja.Value) {
			if _, ok := goja.AssertFunction(fn); ok {
				inst.httpHandler.Store(fn)
			}
		})

		withFetch(inst.eventLoop, vm, fetchConfig{
			client:    nil,
			scheduler: rt.scheduler,
			eventLoop: inst.eventLoop,
		})

		_, err = vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up request script: %w", err)
			return
		}

		inst.running.Store(true)
		setup <- nil
	})
	return setup
}

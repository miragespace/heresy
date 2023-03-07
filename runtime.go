package heresy

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"go.uber.org/zap"
)

type Runtime struct {
	logger    *zap.Logger
	registry  *require.Registry
	scheduler *pond.WorkerPool
	mu        sync.RWMutex
	instance  *runtimeInstance
}

type runtimeInstance struct {
	httpHandler     atomic.Value // goja.Value
	runtimeResolver goja.Callable
	eventLoop       *eventloop.EventLoop
	nativePool      *nativeResolverPool
}

func NewRuntime(logger *zap.Logger) (*Runtime, error) {
	rt := &Runtime{
		logger:    logger,
		registry:  require.NewRegistry(),
		scheduler: pond.New(10, 100),
	}

	return rt, nil
}

func (rt *Runtime) LoadScript(scriptName, script string) error {
	prog, err := goja.Compile(scriptName, script, true)
	if err != nil {
		return fmt.Errorf("error compiling script: %w", err)
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.instance != nil {
		rt.instance.eventLoop.StopNoWait()
	}

	useZapLogger := rt.logger != nil

	eventLoop := eventloop.NewEventLoop(
		eventloop.EnableConsole(!useZapLogger),
		eventloop.WithRegistry(rt.registry),
	)
	eventLoop.Start()

	if useZapLogger {
		runtimeLogger := newRuntimeLogger(scriptName, rt.logger)
		loggerModule := console.RequireWithPrinter(runtimeLogger)
		rt.registry.RegisterNativeModule(loggerModuleName, loggerModule)
		eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			vm.Set("console", require.Require(vm, loggerModuleName))
		})
	}

	instance := &runtimeInstance{
		eventLoop: eventLoop,
	}

	err = <-rt.setupRuntime(prog, instance)
	if err != nil {
		eventLoop.StopNoWait()
		return err
	}

	rt.instance = instance
	return nil
}

func (rt *Runtime) Stop() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.instance != nil {
		rt.instance.eventLoop.StopNoWait()
		rt.instance = nil
	}
	rt.scheduler.Stop()
}

func (rt *Runtime) setupRuntime(prog *goja.Program, inst *runtimeInstance) chan error {
	setup := make(chan error, 1)

	fetch := &fetchConfig{
		eventLoop: inst.eventLoop,
		scheduler: rt.scheduler,
		client: &http.Client{
			Timeout: time.Second * 15,
		},
	}

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		var err error
		_, err = vm.RunProgram(nativePromiseResolverProg)
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

		fetch.Enable(vm)

		_, err = vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up request script: %w", err)
			return
		}

		inst.nativePool = newResolverPool(vm)
		setup <- nil
	})
	return setup
}

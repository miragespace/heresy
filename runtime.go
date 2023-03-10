package heresy

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/modules"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
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
	middlewareHandler atomic.Value                         // goja.Value
	handlerOption     atomic.Pointer[nativeHandlerOptions] // nativeHandlerOptions
	contextPool       *requestContextPool
	eventLoop         *eventloop.EventLoop
	resolver          *promise.PromiseResolver
	stream            *stream.StreamController
	vm                *goja.Runtime
}

func NewRuntime(logger *zap.Logger) (*Runtime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	rt := &Runtime{
		logger:    logger,
		registry:  require.NewRegistryWithLoader(modules.ModulesFS.ReadFile),
		scheduler: pond.New(10, 100),
	}

	return rt, nil
}

func (rt *Runtime) LoadScript(scriptName, script string) (err error) {
	var (
		prog *goja.Program
	)

	prog, err = goja.Compile(scriptName, script, true)
	if err != nil {
		return fmt.Errorf("error compiling script: %w", err)
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

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

	defer func() {
		if err != nil {
			eventLoop.StopNoWait()
		}
	}()

	instance := &runtimeInstance{
		eventLoop: eventLoop,
	}

	err = modules.InjectModules(eventLoop)
	if err != nil {
		return
	}

	instance.resolver, err = promise.NewResolver(eventLoop)
	if err != nil {
		return
	}

	instance.stream, err = stream.NewController(eventLoop)
	if err != nil {
		return
	}

	var options nativeHandlerOptions
	instance.handlerOption.Store(&options)

	err = <-rt.setupRuntime(prog, instance)
	if err != nil {
		return
	}

	if rt.instance != nil {
		rt.instance.eventLoop.StopNoWait()
	}

	// force GC on script reload
	runtime.GC()

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

func (rt *Runtime) setupRuntime(prog *goja.Program, inst *runtimeInstance) (setup chan error) {
	setup = make(chan error, 1)

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		url.Enable(vm)
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Set("registerMiddlewareHandler", func(fn, opt goja.Value) {
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
			}
			if opt == nil {
				return
			}
			var options nativeHandlerOptions
			if err := vm.ExportTo(opt, &options); err == nil {
				inst.handlerOption.Store(&options)
			}
		})

		var err error
		_, err = vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up handler script: %w", err)
			return
		}

		inst.contextPool = newRequestContextPool(inst)
		inst.vm = vm

		setup <- nil
	})

	return
}

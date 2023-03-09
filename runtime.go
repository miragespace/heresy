package heresy

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

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
	runtimeResolver   goja.Callable
	eventLoop         *eventloop.EventLoop
	contextPool       *requestContextPool
}

func NewRuntime(logger *zap.Logger) (*Runtime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	rt := &Runtime{
		logger:    logger,
		registry:  require.NewRegistry(require.WithLoader(modulesFS.ReadFile)),
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

	var options nativeHandlerOptions
	instance.handlerOption.Store(&options)

	err = <-rt.setupRuntime(prog, instance)
	if err != nil {
		eventLoop.StopNoWait()
		return err
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

	resolverProg, err := loadNativePromiseResolver()
	if err != nil {
		setup <- err
		return
	}

	modulesProp, err := loadModulesExporter()
	if err != nil {
		setup <- err
		return
	}

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		url.Enable(vm)

		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

		var err error

		_, err = vm.RunProgram(resolverProg)
		if err != nil {
			setup <- fmt.Errorf("internal error: failed to setting up runtime resolver: %w", err)
			return
		}

		_, err = vm.RunProgram(modulesProp)
		if err != nil {
			setup <- fmt.Errorf("internal error: failed to setting up runtime modules: %w", err)
			return
		}

		runtimeResolver := vm.Get(nativePromiseResolverSymbol)
		resolver, ok := goja.AssertFunction(runtimeResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", nativePromiseResolverSymbol)
			return
		}
		inst.runtimeResolver = resolver

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

		_, err = vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up handler script: %w", err)
			return
		}

		inst.contextPool = newRequestContextPool(inst)

		setup <- nil
	})

	return
}
package heresy

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/polyfill"

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
	scheduler *pond.WorkerPool
	shards    []*instanceShard
	nextShard uint32
	numShards int
}

type instanceShard struct {
	mu       sync.RWMutex
	instance *runtimeInstance
}

type runtimeInstance struct {
	middlewareHandler atomic.Value                         // goja.Value
	eventHandler      atomic.Value                         // goja.Value
	handlerOption     atomic.Pointer[nativeHandlerOptions] // nativeHandlerOptions
	contextPool       *requestContextPool
	eventLoop         *eventloop.EventLoop
	resolver          *promise.PromiseResolver
	stream            *stream.StreamController
	vm                *goja.Runtime

	_testDrainStream goja.Value
}

// NewRuntime returns a new heresy runtime. Use shards > 1 to enable round-robin
// incoming requests to multiple JavaScript runtimes. Recommend not exceeding 4.
func NewRuntime(logger *zap.Logger, shards int) (*Runtime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if shards < 1 {
		return nil, fmt.Errorf("shards cannot be smaller than 1")
	}

	rt := &Runtime{
		logger:    logger,
		scheduler: pond.New(100, 200),
		shards:    make([]*instanceShard, shards),
		numShards: shards,
	}

	for i := range rt.shards {
		rt.shards[i] = &instanceShard{}
	}

	return rt, nil
}

// LoadScript reload the script handling incoming request on-the-fly. Script
// will be executed in a fresh runtime.
func (rt *Runtime) LoadScript(scriptName, script string) (err error) {
	var (
		prog *goja.Program
	)

	prog, err = goja.Compile(scriptName, script, true)
	if err != nil {
		return fmt.Errorf("error compiling script: %w", err)
	}

	// force GC on script reload
	defer runtime.GC()

	for _, shard := range rt.shards {
		shard.mu.Lock()

		registry := require.NewRegistryWithLoader(polyfill.PolyfillFS.ReadFile)

		runtimeLogger := newRuntimeLogger(scriptName, rt.logger)
		loggerModule := console.RequireWithPrinter(runtimeLogger)
		registry.RegisterNativeModule(loggerModuleName, loggerModule)

		instance, err := rt.getInstance(registry)
		if err != nil {
			shard.mu.Unlock()
			return err
		}

		err = <-instance.loadProgram(prog)
		if err != nil {
			return err
		}

		if shard.instance != nil {
			shard.instance.eventLoop.StopNoWait()
		}
		shard.instance = instance

		shard.mu.Unlock()
	}

	return nil
}

func (rt *Runtime) shardRun(fn func(instance *runtimeInstance)) {
	n := atomic.AddUint32(&rt.nextShard, 1)
	shard := rt.shards[(int(n)-1)%rt.numShards]

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	fn(shard.instance)
}

func (rt *Runtime) getInstance(registry *require.Registry) (instance *runtimeInstance, err error) {
	eventLoop := eventloop.NewEventLoop(
		eventloop.EnableConsole(false),
		eventloop.WithRegistry(registry),
	)
	eventLoop.Start()

	defer func() {
		if err != nil {
			eventLoop.StopNoWait()
		}
	}()

	instance = &runtimeInstance{
		eventLoop: eventLoop,
	}

	err = polyfill.PolyfillRuntime(eventLoop)
	if err != nil {
		return
	}

	instance.resolver, err = promise.NewResolver(eventLoop)
	if err != nil {
		return
	}

	instance.stream, err = stream.NewController(eventLoop, rt.scheduler)
	if err != nil {
		return
	}

	var options nativeHandlerOptions
	instance.handlerOption.Store(&options)

	err = <-instance.prepareInstance()

	return
}

func (rt *Runtime) Stop() {
	for _, shard := range rt.shards {
		shard.mu.Lock()
		if shard.instance != nil {
			shard.instance.eventLoop.StopNoWait()
			shard.instance = nil
		}
		shard.mu.Unlock()
	}
	rt.scheduler.Stop()
}

func (inst *runtimeInstance) optionHelper(vm *goja.Runtime, opt goja.Value) {
	if goja.IsUndefined(opt) {
		return
	}
	var options nativeHandlerOptions
	if err := vm.ExportTo(opt, &options); err == nil {
		inst.handlerOption.Store(&options)
	}
}

func (inst *runtimeInstance) prepareInstance() (setup chan error) {
	setup = make(chan error, 1)

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		url.Enable(vm)
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		vm.Set("console", require.Require(vm, loggerModuleName))
		vm.Set("registerMiddlewareHandler", func(fc goja.FunctionCall) (ret goja.Value) {
			ret = goja.Undefined()

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
			}

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			return
		})

		vm.Set("registerEventHandler", func(fc goja.FunctionCall) (ret goja.Value) {
			ret = goja.Undefined()

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.eventHandler.Store(fn)
			}

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			return
		})

		inst.contextPool = newRequestContextPool(inst)
		inst.vm = vm

		setup <- nil
	})

	return
}

func (inst *runtimeInstance) loadProgram(prog *goja.Program) (setup chan error) {
	setup = make(chan error, 1)

	inst.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(prog)
		if err != nil {
			setup <- fmt.Errorf("error setting up handler script: %w", err)
			return
		}

		inst._testDrainStream = vm.Get("drainStream")

		setup <- nil
	})

	return
}

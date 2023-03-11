package heresy

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.miragespace.co/heresy/extensions/fetch"
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

type handlerType int

const (
	handlerTypeUnset handlerType = iota
	handlerTypeExpress
	handlerTypeEvent
)

type runtimeInstance struct {
	middlewareHandler atomic.Value                         // goja.Value
	middlewareType    atomic.Value                         // handlerType
	handlerOption     atomic.Pointer[nativeHandlerOptions] // nativeHandlerOptions
	contextPool       *requestContextPool
	eventPool         *fetchEventPool
	eventLoop         *eventloop.EventLoop
	resolver          *promise.PromiseResolver
	stream            *stream.StreamController
	fetcher           *fetch.Fetcher
	vm                *goja.Runtime
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

	logger.Info("Heresy runtime configured",
		zap.Int("scheduler.maxWorkers", 100),
		zap.Int("shards", shards),
	)

	return rt, nil
}

// LoadScript reload the script handling incoming request on-the-fly. Script
// will be executed in a fresh runtime. Specifying interrupt will interrupt
// currently running VM instead of graceful exit. This is useful when the script
// was misbehaving and needs to be reloaded.
func (rt *Runtime) LoadScript(scriptName, script string, interrupt bool) (err error) {
	var (
		prog *goja.Program
	)

	prog, err = goja.Compile(scriptName, script, true)
	if err != nil {
		return fmt.Errorf("error compiling script: %w", err)
	}

	// force GC on script reload
	defer runtime.GC()

	start := time.Now()
	for _, shard := range rt.shards {
		registry := require.NewRegistryWithLoader(polyfill.PolyfillFS.ReadFile)

		runtimeLogger := newRuntimeLogger(scriptName, rt.logger)
		loggerModule := console.RequireWithPrinter(runtimeLogger)
		registry.RegisterNativeModule(loggerModuleName, loggerModule)

		instance, err := rt.getInstance(registry)
		if err != nil {
			return err
		}

		err = <-instance.loadProgram(prog)
		if err != nil {
			return err
		}

		shard.mu.Lock()
		if shard.instance != nil {
			shard.instance.stop(interrupt)
		}
		shard.instance = instance
		shard.mu.Unlock()
	}

	duration := time.Since(start)
	rt.logger.Info("All shards reloaded",
		zap.Duration("duration", duration),
		zap.String("script", scriptName),
		zap.Int("shards", rt.numShards),
	)

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

	instance.fetcher, err = fetch.NewFetcher(fetch.FetcherConfig{
		Eventloop: eventLoop,
		Stream:    instance.stream,
		Scheduler: rt.scheduler,
		Client: &http.Client{
			Timeout: time.Second * 10,
		},
	})
	if err != nil {
		return
	}

	var options nativeHandlerOptions
	instance.handlerOption.Store(&options)
	instance.middlewareType.Store(handlerTypeUnset)

	err = <-instance.prepareInstance()

	return
}

func (rt *Runtime) Stop(interrupt bool) {
	for _, shard := range rt.shards {
		shard.mu.Lock()
		if shard.instance != nil {
			shard.instance.stop(interrupt)
			shard.instance = nil
		}
		shard.mu.Unlock()
	}
	rt.scheduler.Stop()
}

func (inst *runtimeInstance) stop(interrupt bool) {
	if interrupt {
		inst.vm.Interrupt(context.Canceled)
	}
	inst.eventLoop.StopNoWait()
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

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
				inst.middlewareType.Store(handlerTypeExpress)
			}

			return
		})

		vm.Set("registerEventHandler", func(fc goja.FunctionCall) (ret goja.Value) {
			ret = goja.Undefined()

			opt := fc.Argument(1)
			inst.optionHelper(vm, opt)

			fn := fc.Argument(0)
			if _, ok := goja.AssertFunction(fn); ok {
				inst.middlewareHandler.Store(fn)
				inst.middlewareType.Store(handlerTypeEvent)
			}

			return
		})

		inst.contextPool = newRequestContextPool(inst)
		inst.eventPool = newFetchEventPool(inst)
		inst.vm = vm // reference is kept for .Interrupt

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
		setup <- nil
	})

	return
}

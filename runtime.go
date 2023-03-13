package heresy

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
	"go.miragespace.co/heresy/extensions/zap_console"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"go.uber.org/zap"
	"golang.org/x/sys/cpu"
)

type Runtime struct {
	logger    *zap.Logger
	transport http.RoundTripper
	shards    []atomic.Pointer[runtimeInstance]
	_         cpu.CacheLinePad
	nextShard uint32
	_         cpu.CacheLinePad
	numShards int
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

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 500
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 10
	t.IdleConnTimeout = time.Minute

	rt := &Runtime{
		logger:    logger,
		transport: t,
		shards:    make([]atomic.Pointer[runtimeInstance], shards),
		numShards: shards,
	}

	for i := range rt.shards {
		rt.shards[i] = atomic.Pointer[runtimeInstance]{}
		rt.shards[i].Store(nilInstance)
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
	for i := range rt.shards {
		registry := require.NewRegistryWithLoader(polyfill.PolyfillFS.ReadFile)

		loggerModule := zap_console.RequireWithLogger(rt.logger)
		registry.RegisterNativeModule(zap_console.ModuleName, loggerModule)

		instance, err := rt.getInstance(rt.transport, registry)
		if err != nil {
			return err
		}

		err = <-instance.loadProgram(prog)
		if err != nil {
			return err
		}

		old := rt.shards[i].Swap(instance)
		if old != nilInstance {
			old.stop(interrupt)
		}
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
	instance := rt.shards[int(n)%rt.numShards].Load()

	fn(instance)
}

func (rt *Runtime) getInstance(t http.RoundTripper, registry *require.Registry) (instance *runtimeInstance, err error) {
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
		logger:        rt.logger,
		eventLoop:     eventLoop,
		ioContextPool: common.NewIOContextPool(rt.logger, 10),
	}

	var options nativeHandlerOptions
	instance.handlerOption.Store(&options)
	instance.middlewareType.Store(handlerTypeUnset)

	var symbols *polyfill.RuntimeSymbols
	symbols, err = polyfill.PolyfillRuntime(eventLoop)
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

	instance.fetcher, err = fetch.NewFetch(fetch.FetchConfig{
		Eventloop: eventLoop,
		Stream:    instance.stream,
		Client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: t,
		},
	})
	if err != nil {
		return
	}

	err = <-instance.prepareInstance(rt.logger, symbols)

	return
}

func (rt *Runtime) Stop(interrupt bool) {
	for i := range rt.shards {
		old := rt.shards[i].Swap(nilInstance)
		if old != nilInstance {
			old.stop(interrupt)
		}
	}
}

package heresy

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"go.miragespace.co/heresy/extensions/console"
	"go.miragespace.co/heresy/extensions/fetch"
	"go.miragespace.co/heresy/extensions/kv"
	"go.miragespace.co/heresy/extensions/promise"
	"go.miragespace.co/heresy/extensions/stream"
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
	kvManager *kv.KVManager
	shards    []atomic.Pointer[runtimeInstance]
	_         cpu.CacheLinePad
	nextShard uint32
	_         cpu.CacheLinePad
	numShards int
}

// NewRuntime returns a new heresy runtime. Use shards > 1 to enable round-robin
// incoming requests to multiple JavaScript runtimes. Recommend not exceeding 4.
func NewRuntime(logger *zap.Logger, kvManager *kv.KVManager, shards int) (*Runtime, error) {
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
		kvManager: kvManager,
		transport: t,
		shards:    make([]atomic.Pointer[runtimeInstance], shards),
		numShards: shards,
	}

	for i := range rt.shards {
		rt.shards[i] = atomic.Pointer[runtimeInstance]{}
		rt.shards[i].Store(nilInstance)
	}

	atomic.AddUint32(&rt.nextShard, ^uint32(0))

	logger.Info("Heresy runtime configured",
		zap.Int("io.outbound", 10),
		zap.Int("runtime.shards", shards),
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

		loggerModule := console.RequireWithLogger(rt.logger)
		registry.RegisterNativeModule(console.ModuleName, loggerModule)

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

func (rt *Runtime) shardRun(fn func(index int, instance *runtimeInstance)) {
	n := atomic.AddUint32(&rt.nextShard, 1)
	i := int(n) % rt.numShards
	instance := rt.shards[i].Load()

	fn(i, instance)
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
		logger:    rt.logger,
		kv:        rt.kvManager,
		eventLoop: eventLoop,
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

	instance.stream, err = stream.NewController(eventLoop, symbols)
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

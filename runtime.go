package heresy

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
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
	reloadMu   sync.Mutex
	shardMu    sync.RWMutex
	generation uint64
	instances  map[*runtimeInstance]struct{}
	logger     *zap.Logger
	transport  http.RoundTripper
	kvManager  *kv.KVManager
	shards     []atomic.Pointer[runtimeInstance]
	_          cpu.CacheLinePad
	nextShard  uint32
	_          cpu.CacheLinePad
	numShards  int
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
	if kvManager == nil {
		kvManager = kv.NewKVManager()
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
		instances: make(map[*runtimeInstance]struct{}),
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
	rt.reloadMu.Lock()
	defer rt.reloadMu.Unlock()
	rt.shardMu.RLock()
	generation := rt.generation
	rt.shardMu.RUnlock()

	// force GC on script reload
	defer runtime.GC()

	start := time.Now()
	prepared := make([]*runtimeInstance, 0, rt.numShards)
	defer func() {
		if err != nil {
			for _, instance := range prepared {
				instance.stop(true)
			}
		}
	}()
	for range rt.shards {
		registry := require.NewRegistryWithLoader(polyfill.PolyfillFS.ReadFile)

		loggerModule := console.RequireWithLogger(rt.logger)
		registry.RegisterNativeModule(console.ModuleName, loggerModule)

		instance, err := rt.getInstance(rt.transport, registry)
		if err != nil {
			return err
		}
		prepared = append(prepared, instance)
		rt.shardMu.Lock()
		if rt.generation != generation {
			rt.shardMu.Unlock()
			return ErrRuntimeNotReady
		}
		rt.instances[instance] = struct{}{}
		rt.shardMu.Unlock()
		rt.forgetWhenStopped(instance)

		err = <-instance.loadProgram(prog)
		if err != nil {
			return err
		}
	}
	// Publish a complete generation and acquire request leases under the same lock.
	rt.shardMu.Lock()
	if rt.generation != generation {
		rt.shardMu.Unlock()
		return ErrRuntimeNotReady
	}
	previous := make([]*runtimeInstance, len(rt.shards))
	for i, instance := range prepared {
		previous[i] = rt.shards[i].Swap(instance)
	}
	rt.shardMu.Unlock()
	for _, old := range previous {
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
	i := int(uint64(n) % uint64(rt.numShards))
	rt.shardMu.RLock()
	instance := rt.shards[i].Load()
	if instance != nilInstance {
		instance.active.Add(1)
	}
	rt.shardMu.RUnlock()

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
			if instance != nil {
				instance.cancel()
			}
			eventLoop.Terminate()
		}
	}()

	instance = &runtimeInstance{
		logger:    rt.logger,
		kv:        rt.kvManager,
		eventLoop: eventLoop,
		stopped:   make(chan struct{}),
	}
	instance.context, instance.cancel = context.WithCancel(context.Background())

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
	rt.shardMu.Lock()
	rt.generation++
	for i := range rt.shards {
		rt.shards[i].Store(nilInstance)
	}
	instances := make([]*runtimeInstance, 0, len(rt.instances))
	for instance := range rt.instances {
		instances = append(instances, instance)
	}
	rt.shardMu.Unlock()
	for _, instance := range instances {
		instance.stop(interrupt)
	}
	if interrupt {
		for _, instance := range instances {
			instance.active.Wait()
		}
	}
	if transport, ok := rt.transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}

func (rt *Runtime) forgetWhenStopped(instance *runtimeInstance) {
	go func() {
		<-instance.stopped
		instance.active.Wait()
		rt.shardMu.Lock()
		delete(rt.instances, instance)
		rt.shardMu.Unlock()
	}()
}

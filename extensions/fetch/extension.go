package fetch

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.miragespace.co/heresy/extensions/stream"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const UserAgent = "heresy-runtime/fetcher"

type Fetcher struct {
	FetcherConfig
	runtimeWrapper goja.Callable
	nativeObjPool  sync.Pool
}

type FetcherConfig struct {
	Eventloop *eventloop.EventLoop
	Stream    *stream.StreamController
	Scheduler *pond.WorkerPool
	Client    *http.Client
}

func (c *FetcherConfig) Validate() error {
	if c.Eventloop == nil {
		return fmt.Errorf("nil Eventloop is invalid")
	}
	if c.Stream == nil {
		return fmt.Errorf("nil StreamController is invalid")
	}
	if c.Scheduler == nil {
		return fmt.Errorf("nil Scheduler is invalid")
	}
	if c.Client == nil {
		return fmt.Errorf("nil http.Client is invalid")
	}
	return nil
}

func NewFetcher(cfg FetcherConfig) (*Fetcher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	f := &Fetcher{
		FetcherConfig: cfg,
	}

	setup := make(chan error, 1)
	f.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(fetchWrapperProg)
		if err != nil {
			setup <- err
			return
		}

		promiseResolver := vm.Get(fetchWrapperSymbol)
		wrapper, ok := goja.AssertFunction(promiseResolver)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", fetchWrapperSymbol)
			return
		}
		f.runtimeWrapper = wrapper

		f.nativeObjPool = sync.Pool{
			New: func() any {
				fetch := &nativeFetchWrapper{
					cfg: f.FetcherConfig,
				}
				fetch._doFetch = vm.ToValue(fetch.DoFetch)
				return vm.NewDynamicObject(fetch)
			},
		}

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Fetcher) NewFetch(ctx context.Context) (goja.Value, error) {
	valCh := make(chan goja.Value, 1)
	errCh := make(chan error, 1)
	f.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
		s, err := f.NewFetchVM(ctx, vm)
		if err != nil {
			errCh <- err
		} else {
			valCh <- s
		}
	})

	select {
	case err := <-errCh:
		return nil, err
	case v := <-valCh:
		return v, nil
	}
}

func (f *Fetcher) NewFetchVM(ctx context.Context, vm *goja.Runtime) (goja.Value, error) {
	obj := f.nativeObjPool.Get().(*goja.Object)
	fetch := obj.Export().(*nativeFetchWrapper)
	fetch.ctx = ctx

	return f.runtimeWrapper(goja.Undefined(), obj)
}

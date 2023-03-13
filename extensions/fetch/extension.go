package fetch

import (
	"fmt"
	"net/http"
	"sync"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/stream"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const UserAgent = "heresy-runtime/fetcher"

var ErrUnsupportedReadableStream = fmt.Errorf("using custom ReadableStream as body is currently unsupported")

type Fetch struct {
	FetchConfig
	runtimeFetchWrapper  goja.Callable
	runtimeReponseHelper goja.Value
	fetcherPool          sync.Pool
	respPool             *responseProxyPool
}

type FetchConfig struct {
	Stream    *stream.StreamController
	Eventloop *eventloop.EventLoop
	Client    *http.Client
}

type NativeFetcher struct {
	nativeWrapper *NativeFetchWrapper
	nativeFunc    goja.Value
}

func (n *NativeFetcher) NativeFunc() goja.Value {
	return n.nativeFunc
}

func (c *FetchConfig) Validate() error {
	if c.Eventloop == nil {
		return fmt.Errorf("nil Eventloop is invalid")
	}
	if c.Stream == nil {
		return fmt.Errorf("nil StreamController is invalid")
	}
	if c.Client == nil {
		return fmt.Errorf("nil http.Client is invalid")
	}
	return nil
}

func NewFetch(cfg FetchConfig) (*Fetch, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	f := &Fetch{
		FetchConfig: cfg,
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
		f.runtimeFetchWrapper = wrapper

		promiseResolver = vm.Get(responseHelperSymbol)
		f.runtimeReponseHelper = promiseResolver

		f.respPool = newResponseProxyPool(vm, f.Stream)

		f.fetcherPool = sync.Pool{
			New: func() any {
				wrapper := &NativeFetchWrapper{
					cfg: f.FetchConfig,
				}
				obj := vm.CreateObject(nil)
				obj.Set("doFetch", vm.ToValue(wrapper.DoFetch))
				fn, err := f.runtimeFetchWrapper(goja.Undefined(), obj)
				if err != nil {
					panic(fmt.Errorf("runtime panic: Failed to get native fetch: %w", err))
				}
				return &NativeFetcher{
					nativeWrapper: wrapper,
					nativeFunc:    fn,
				}
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

func (f *Fetch) GetResponseHelper() goja.Value {
	return f.runtimeReponseHelper
}

// func (f *Fetch) NewNativeFetch(t *common.IOContext) *NativeFetcher {
// 	valCh := make(chan *NativeFetcher, 1)
// 	f.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
// 		valCh <- f.NewNativeFetchVM(t, vm)
// 	})

// 	return <-valCh
// }

func (f *Fetch) NewNativeFetchVM(t *common.IOContext, vm *goja.Runtime) *NativeFetcher {
	fetcher := f.fetcherPool.Get().(*NativeFetcher)
	fetcher.nativeWrapper.ioContext = t
	fetcher.nativeWrapper.respPool = f.respPool

	t.RegisterCleanup(func() {
		fetcher.nativeWrapper.respPool = nil
		fetcher.nativeWrapper.ioContext = nil
		f.fetcherPool.Put(fetcher)
	})

	return fetcher
}

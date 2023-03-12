package fetch

import (
	"context"
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
	nativeObjPool        sync.Pool
}

type FetchConfig struct {
	Stream    *stream.StreamController
	Eventloop *eventloop.EventLoop
	Client    *http.Client
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

		f.nativeObjPool = sync.Pool{
			New: func() any {
				w := &nativeFetchWrapper{
					cfg: f.FetchConfig,
				}
				w._doFetch = vm.ToValue(w.DoFetch)
				w._unsetCtx = vm.ToValue(w.UnsetCtx)
				obj := vm.NewDynamicObject(w)
				fn, _ := f.runtimeFetchWrapper(goja.Undefined(), obj)
				return fn
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

func (f *Fetch) DoneWith(native goja.Value) {
	if _, ok := goja.AssertFunction(native); !ok {
		return
	}
	f.nativeObjPool.Put(native)
}

func (f *Fetch) NewNativeFetch(ctx context.Context) (goja.Value, error) {
	valCh := make(chan goja.Value, 1)
	errCh := make(chan error, 1)
	f.Eventloop.RunOnLoop(func(vm *goja.Runtime) {
		s, err := f.NewNativeFetchVM(ctx, vm)
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

func (f *Fetch) NewNativeFetchVM(ctx context.Context, vm *goja.Runtime) (goja.Value, error) {
	fn := f.nativeObjPool.Get().(*goja.Object)
	w := fn.Get("wrapper").Export().(*nativeFetchWrapper)
	w.WithContext(ctx)

	return fn, nil
}

func AsNativeWrapper(wrapper goja.Value) (*common.NativeReaderWrapper, bool) {
	if wrapper == nil {
		return nil, false
	}
	w, ok := wrapper.Export().(*common.NativeReaderWrapper)
	return w, ok
}

package stream

import (
	"fmt"
	"io"
	"sync"

	"go.miragespace.co/heresy/extensions/common"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type StreamController struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
	nativeObjPool  sync.Pool
	vm             *goja.Runtime
}

func NewController(eventLoop *eventloop.EventLoop) (*StreamController, error) {
	t := &StreamController{
		eventLoop: eventLoop,
	}

	setup := make(chan error, 1)
	eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		_, err := vm.RunProgram(streamWrapperProg)
		if err != nil {
			setup <- err
			return
		}

		runtimeIOWrapper := vm.Get(streamWrapperSymbol)
		wrapper, ok := goja.AssertFunction(runtimeIOWrapper)
		if !ok {
			setup <- fmt.Errorf("internal error: %s is not a function", streamWrapperSymbol)
			return
		}
		t.runtimeWrapper = wrapper
		t.vm = vm

		t.nativeObjPool = sync.Pool{
			New: func() any {
				w := common.NewNativeReaderWrapper(vm, eventLoop)
				w.OverwriteClose(vm.ToValue(t.closeReader))
				return w
			},
		}

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return t, nil
}

// called if .close() is called from JavaScript
func (s *StreamController) closeReader(fc goja.FunctionCall) goja.Value {
	w := fc.Argument(0).Export().(*common.NativeReaderWrapper)
	s.Close(w)
	return goja.Undefined()
}

// called if external module took control of the io.Reader,
// such as Fetcher
func (s *StreamController) Close(w *common.NativeReaderWrapper) {
	if !w.SameRuntime(s.vm) {
		return
	}
	w.Close()
	w.WithReader(nil)
	s.nativeObjPool.Put(w)
}

func (s *StreamController) NewReadableStream(r io.ReadCloser) (goja.Value, error) {
	valCh := make(chan goja.Value, 1)
	errCh := make(chan error, 1)
	s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
		s, err := s.NewReadableStreamVM(r, vm)
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

func (s *StreamController) NewReadableStreamVM(r io.ReadCloser, vm *goja.Runtime) (goja.Value, error) {
	w := s.nativeObjPool.Get().(*common.NativeReaderWrapper)
	w.WithReader(r)

	return s.runtimeWrapper(goja.Undefined(), w.NativeObject())
}

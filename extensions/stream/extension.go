package stream

import (
	"fmt"
	"io"
	"sync"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const BufferSize = 8 * 1024

type StreamController struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
	nativeObjPool  sync.Pool
}

func NewController(eventLoop *eventloop.EventLoop, scheduler *pond.WorkerPool) (*StreamController, error) {
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

		t.nativeObjPool = sync.Pool{
			New: func() any {
				w := &nativeReaderWrapper{
					eventLoop: eventLoop,
					scheduler: scheduler,
				}
				w._readInto = vm.ToValue(w.ReadInto)
				w._size = vm.ToValue(BufferSize)
				w._close = vm.ToValue(t.closeReader)
				return vm.NewDynamicObject(w)
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

func (s *StreamController) closeReader(fc goja.FunctionCall) goja.Value {
	obj := fc.Argument(0).(*goja.Object)
	w := obj.Export().(*nativeReaderWrapper)
	if w.reader != nil {
		w.reader.Close()
		w.reader = nil
		s.nativeObjPool.Put(obj)
	}
	return goja.Undefined()
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
	obj := s.nativeObjPool.Get().(*goja.Object)
	w := obj.Export().(*nativeReaderWrapper)
	w.reader = r

	return s.runtimeWrapper(goja.Undefined(), obj)
}

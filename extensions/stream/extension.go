package stream

import (
	"fmt"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type StreamController struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
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

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return t, nil
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
	w := &nativeReaderWrapper{
		reader:    r,
		eventLoop: s.eventLoop,
	}

	return s.runtimeWrapper(goja.Undefined(), vm.ToValue(w))
}

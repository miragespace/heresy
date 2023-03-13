package stream

import (
	"fmt"
	"io"
	"sync"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/shared"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type StreamController struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
	streamPool     sync.Pool
	respPool       sync.Pool
}

type ReadableStream struct {
	nativeWrapper *shared.NativeReaderWrapper
	nativeStream  goja.Value
}

func (r *ReadableStream) NativeStream() goja.Value {
	return r.nativeStream
}

func NewController(eventLoop *eventloop.EventLoop) (*StreamController, error) {
	s := &StreamController{
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
		s.runtimeWrapper = wrapper

		s.streamPool = sync.Pool{
			New: func() any {
				wrapper := shared.NewNativeReaderWrapper(vm, s.eventLoop)
				fn, err := s.runtimeWrapper(goja.Undefined(), wrapper.NativeObject())
				if err != nil {
					panic(fmt.Errorf("runtime panic: Failed to get native ReadableStream: %w", err))
				}

				return &ReadableStream{
					nativeWrapper: wrapper,
					nativeStream:  fn,
				}
			},
		}

		s.respPool = sync.Pool{
			New: func() any {
				return newResponseProxy(vm, s)
			},
		}

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *StreamController) GetResponseProxy(t *common.IOContext) *ResponseProxy {
	resp := s.respPool.Get().(*ResponseProxy)
	t.RegisterCleanup(func() {
		s.respPool.Put(resp)
	})
	return resp
}

// func (s *StreamController) NewReadableStream(t *common.IOContext, r io.ReadCloser) *ReadableStream {
// 	valCh := make(chan *ReadableStream, 1)
// 	s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
// 		s := s.NewReadableStreamVM(t, r, vm)
// 		valCh <- s
// 	})

// 	return <-valCh
// }

func (s *StreamController) NewReadableStreamVM(t *common.IOContext, r io.ReadCloser, vm *goja.Runtime) *ReadableStream {
	stream := s.streamPool.Get().(*ReadableStream)
	stream.nativeWrapper.WithReader(r)
	t.TrackReader(stream.nativeWrapper)

	t.RegisterCleanup(func() {
		s.streamPool.Put(stream)
	})

	return stream
}

func AssertReader(native goja.Value, vm *goja.Runtime) (io.Reader, bool) {
	obj := native.ToObject(vm)
	wrapper := obj.Get("wrapper")
	if wrapper == nil {
		return nil, false
	}
	w, ok := wrapper.Export().(*shared.NativeReaderWrapper)
	if ok {
		return w.Reader(), true
	}
	return nil, false
}

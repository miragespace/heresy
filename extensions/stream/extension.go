package stream

import (
	"expvar"
	"fmt"
	"io"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/shared"
	"go.miragespace.co/heresy/extensions/common/x"
	"go.miragespace.co/heresy/polyfill"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

var (
	wrapperNew = expvar.NewInt("readerWrapper.New")
	wrapperPut = expvar.NewInt("readerWrapper.Put")
	respNew    = expvar.NewInt("responseProxy.New")
	respPut    = expvar.NewInt("responseProxy.Put")
)

type StreamController struct {
	eventLoop      *eventloop.EventLoop
	runtimeWrapper goja.Callable
	streamPool     *x.Pool[*ReadableStream]
	respPool       *x.Pool[*ResponseProxy]
}

type ReadableStream struct {
	nativeWrapper *NativeReaderWrapper
	nativeStream  goja.Value
}

func (r *ReadableStream) NativeStream() goja.Value {
	return r.nativeStream
}

func NewController(eventLoop *eventloop.EventLoop, symbols *polyfill.RuntimeSymbols) (*StreamController, error) {
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

		s.streamPool = x.NewPool[*ReadableStream](x.DefaultPoolCapacity).
			WithFactory(func() *ReadableStream {
				wrapperNew.Add(1)
				wrapper := NewNativeReaderWrapper(vm, s.eventLoop)
				return &ReadableStream{
					nativeWrapper: wrapper,
				}
			})

		s.respPool = x.NewPool[*ResponseProxy](x.DefaultPoolCapacity).
			WithFactory(func() *ResponseProxy {
				respNew.Add(1)
				return newResponseProxy(vm, s, symbols)
			})

		setup <- nil
	})

	err := <-setup
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *StreamController) GetResponseProxy(t *common.IOContext) *ResponseProxy {
	resp := s.respPool.Get()
	t.RegisterCleanup(func() {
		s.respPool.Put(resp)
		respPut.Add(1)
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
	stream := s.streamPool.Get()
	stream.nativeWrapper.WithReader(r)

	// unfortunately, ReadableStream itself cannot be reused. we have to create one every time.
	fn, err := s.runtimeWrapper(goja.Undefined(), stream.nativeWrapper.NativeObject())
	if err != nil {
		panic(fmt.Errorf("runtime panic: Failed to get native ReadableStream: %w", err))
	}
	stream.nativeStream = fn

	t.RegisterCleanup(func() {
		buf := shared.GetBuffer()
		defer shared.PutBuffer(buf)

		stream.nativeWrapper.Reset(buf)
		stream.nativeStream = nil
		s.streamPool.Put(stream)
		wrapperPut.Add(1)
	})

	return stream
}

func AssertReader(native goja.Value, vm *goja.Runtime) (io.Reader, bool) {
	obj := native.ToObject(vm)
	wrapper := obj.Get("wrapper")
	if wrapper == nil {
		return nil, false
	}
	w, ok := wrapper.Export().(*NativeReaderWrapper)
	if ok {
		return w.Reader(), true
	}
	return nil, false
}

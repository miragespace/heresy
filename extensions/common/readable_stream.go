package common

import (
	"errors"
	"io"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const BufferSize = 16 * 1024

type NativeReaderWrapper struct {
	reader    io.ReadCloser
	eventLoop *eventloop.EventLoop
	scheduler *pond.WorkerPool
	nativeObj *goja.Object
	vm        *goja.Runtime

	_readInto goja.Value
	_size     goja.Value
	_close    goja.Value
}

var keys = []string{"bufferSize"}

var _ goja.DynamicObject = (*NativeReaderWrapper)(nil)

func NewNativeReaderWrapper(vm *goja.Runtime, eventLoop *eventloop.EventLoop, scheduler *pond.WorkerPool) *NativeReaderWrapper {
	s := &NativeReaderWrapper{
		eventLoop: eventLoop,
		scheduler: scheduler,
		vm:        vm,
	}
	s.nativeObj = vm.NewDynamicObject(s)
	s._readInto = vm.ToValue(s.ReadInto)
	s._size = vm.ToValue(BufferSize)
	s._close = vm.ToValue(s.nativeClose)
	return s
}

func (s *NativeReaderWrapper) WithReader(r io.ReadCloser) {
	s.reader = r
}

func (s *NativeReaderWrapper) GetReader() io.ReadCloser {
	return s.reader
}

func (s *NativeReaderWrapper) OverwriteClose(fn goja.Value) {
	s._close = fn
}

func (s *NativeReaderWrapper) NativeObject() goja.Value {
	return s.nativeObj
}

func (s *NativeReaderWrapper) SameRuntime(vm *goja.Runtime) bool {
	return s.vm == vm
}

// Get a property value for the key. May return nil if the property does not exist.
func (s *NativeReaderWrapper) Get(key string) goja.Value {
	switch key {
	case "readInto":
		return s._readInto
	case "bufferSize":
		return s._size
	case "close":
		return s._close
	default:
		return goja.Undefined()
	}
}

func (s *NativeReaderWrapper) Set(key string, val goja.Value) bool {
	return false
}

func (s *NativeReaderWrapper) Has(key string) bool {
	return !goja.IsUndefined(s.Get(key))
}

func (s *NativeReaderWrapper) Delete(key string) bool {
	return false
}

func (s *NativeReaderWrapper) Keys() []string {
	return keys
}

func (s *NativeReaderWrapper) ReadInto(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	promise, resolve, reject := vm.NewPromise()
	ret = vm.ToValue(promise)

	var (
		viewBuffer                  = fc.Argument(0)
		byteOffset                  = fc.Argument(1)
		byteLength                  = fc.Argument(2)
		buffer     goja.ArrayBuffer = viewBuffer.Export().(goja.ArrayBuffer)
		offset     int64            = byteOffset.ToInteger()
		length     int64            = byteLength.ToInteger()
	)

	buf := buffer.Bytes()[offset:length]
	s.scheduler.Submit(func() {
		n, err := s.reader.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			s.reader.Close()
			s.eventLoop.RunOnLoop(func(*goja.Runtime) {
				reject(err)
			})
		} else {
			s.eventLoop.RunOnLoop(func(*goja.Runtime) {
				resolve(n)
			})
		}
	})

	return
}

func (s *NativeReaderWrapper) Close() {
	if s.reader != nil {
		s.reader.Close()
	}
}

func (s *NativeReaderWrapper) nativeClose(fc goja.FunctionCall) goja.Value {
	s.Close()
	return goja.Undefined()
}

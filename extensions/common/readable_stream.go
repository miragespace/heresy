package common

import (
	"errors"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const BufferSize = 8 * 1024

type NativeReaderWrapper struct {
	reader    io.ReadCloser
	eventLoop *eventloop.EventLoop
	nativeObj *goja.Object
	vm        *goja.Runtime

	_readInto goja.Value
	_size     goja.Value
}

var keys = []string{"bufferSize"}

var _ goja.DynamicObject = (*NativeReaderWrapper)(nil)

func NewNativeReaderWrapper(vm *goja.Runtime, eventLoop *eventloop.EventLoop) *NativeReaderWrapper {
	s := &NativeReaderWrapper{
		eventLoop: eventLoop,
		vm:        vm,
	}
	s.nativeObj = vm.NewDynamicObject(s)
	s._readInto = vm.ToValue(s.ReadInto)
	s._size = vm.ToValue(BufferSize)
	return s
}

func (s *NativeReaderWrapper) WithReader(r io.ReadCloser) {
	s.reader = r
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
	go func() {
		n, err := s.reader.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			s.reader.Close()
			s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				reject(vm.NewGoError(err))
			})
		} else {
			s.eventLoop.RunOnLoop(func(*goja.Runtime) {
				resolve(n)
			})
		}
	}()

	return
}

func AssertReader(native goja.Value, vm *goja.Runtime) (io.Reader, bool) {
	// see extension/stream/wrapper.ts
	obj := native.ToObject(vm)
	wrapper := obj.Get("wrapper")
	if wrapper == nil {
		return nil, false
	}
	w, ok := wrapper.Export().(*NativeReaderWrapper)
	if ok {
		return w.reader, true
	}
	return nil, false
}

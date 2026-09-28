package stream

import (
	"context"
	"errors"
	"io"

	"go.miragespace.co/heresy/extensions/common"
	"go.miragespace.co/heresy/extensions/common/shared"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type NativeReaderWrapper struct {
	ioContext *common.IOContext
	reader    io.ReadCloser
	eventLoop *eventloop.EventLoop
	nativeObj *goja.Object
	vm        *goja.Runtime

	_readInto goja.Value
	_size     goja.Value
}

var readerWrapperKeys = []string{"bufferSize"}

var _ goja.DynamicObject = (*NativeReaderWrapper)(nil)

func NewNativeReaderWrapper(vm *goja.Runtime, eventLoop *eventloop.EventLoop) *NativeReaderWrapper {
	s := &NativeReaderWrapper{
		eventLoop: eventLoop,
		vm:        vm,
	}
	s.nativeObj = vm.NewDynamicObject(s)
	s._readInto = vm.ToValue(s.readInto)
	s._size = vm.ToValue(shared.BufferSize)
	return s
}

func (s *NativeReaderWrapper) Reset(buf []byte) {
	s.reader.Close()
	s.reader = nil
	s.ioContext = nil
}

func (s *NativeReaderWrapper) Reader() io.ReadCloser {
	return s.reader
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
	for _, k := range readerWrapperKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (s *NativeReaderWrapper) Delete(key string) bool {
	return false
}

func (s *NativeReaderWrapper) Keys() []string {
	return readerWrapperKeys
}

func (s *NativeReaderWrapper) readInto(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
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

	if offset < 0 || length < 0 || offset > int64(len(buffer.Bytes())) || length > int64(len(buffer.Bytes()))-offset {
		panic(vm.NewTypeError("invalid read buffer range"))
	}
	// The polyfill does not detach BYOB buffers. Copy back on the VM thread so
	// user JavaScript cannot race a native read into the same ArrayBuffer.
	buf := make([]byte, length)
	t := s.ioContext
	reader := s.reader
	t.StartIO()
	go func() {
		stop := context.AfterFunc(t.Context(), func() { reader.Close() })
		n, err := reader.Read(buf)
		stop()
		if err != nil && !errors.Is(err, io.EOF) {
			reader.Close()
			t.CompleteIO(s.eventLoop, func(vm *goja.Runtime) {
				reject(vm.NewGoError(err))
			}, nil)
		} else {
			t.CompleteIO(s.eventLoop, func(*goja.Runtime) {
				copy(buffer.Bytes()[offset:offset+int64(n)], buf[:n])
				resolve(n)
			}, nil)
		}
	}()

	return
}

package stream

import (
	"errors"
	"io"

	"github.com/alitto/pond"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type nativeReaderWrapper struct {
	reader    io.ReadCloser
	eventLoop *eventloop.EventLoop
	scheduler *pond.WorkerPool

	_readInto goja.Value
	_size     goja.Value
	_close    goja.Value
}

var keys = []string{"readInto", "bufferSize", "length"}

var _ goja.DynamicObject = (*nativeReaderWrapper)(nil)

// Get a property value for the key. May return nil if the property does not exist.
func (s *nativeReaderWrapper) Get(key string) goja.Value {
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

// Set a property value for the key. Return true if success, false otherwise.
func (s *nativeReaderWrapper) Set(key string, val goja.Value) bool {
	return false
}

// Has should return true if and only if the property exists.
func (s *nativeReaderWrapper) Has(key string) bool {
	return !goja.IsUndefined(s.Get(key))
}

// Delete the property for the key. Returns true on success (note, that includes missing property).
func (s *nativeReaderWrapper) Delete(key string) bool {
	return false
}

// Keys returns a list of all existing property keys. There are no checks for duplicates or to make sure
// that the order conforms to https://262.ecma-international.org/#sec-ordinaryownpropertykeys
func (s *nativeReaderWrapper) Keys() []string {
	return keys
}

func (s *nativeReaderWrapper) ReadInto(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
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

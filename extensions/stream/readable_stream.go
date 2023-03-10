package stream

import (
	"errors"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

const BufferSize = 8 * 1024

type nativeReaderWrapper struct {
	reader    io.ReadCloser
	eventLoop *eventloop.EventLoop
}

func (s *nativeReaderWrapper) ReadInto(fc goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	promise, resolve, reject := vm.NewPromise()
	ret = vm.ToValue(promise)

	var (
		err        error
		buffer     goja.ArrayBuffer
		offset     int
		length     int
		viewBuffer = fc.Argument(0)
		byteOffset = fc.Argument(1)
		byteLength = fc.Argument(2)
	)

	err = vm.ExportTo(viewBuffer, &buffer)
	if err != nil {
		s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			reject(err)
		})
		return
	}

	err = vm.ExportTo(byteOffset, &offset)
	if err != nil {
		s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			reject(err)
		})
		return
	}

	err = vm.ExportTo(byteLength, &length)
	if err != nil {
		s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			reject(err)
		})
		return
	}

	buf := buffer.Bytes()[offset:length]
	go func() {
		n, err := s.reader.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				reject(err)
			})
		} else {
			s.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
				resolve(n)
			})
		}
	}()

	return
}

func (s *nativeReaderWrapper) Size(fn goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(BufferSize)
}

func (s *nativeReaderWrapper) Close(fc goja.FunctionCall) goja.Value {
	s.reader.Close()
	return goja.Undefined()
}

package kv

import (
	"errors"

	"go.miragespace.co/heresy/extensions/common"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

type NativeKVProxy struct {
	ioContext *common.IOContext
	backing   KV
	nativeGet goja.Value
	nativePut goja.Value
	nativeDel goja.Value
	nativeObj *goja.Object
	vm        *goja.Runtime
	eventLoop *eventloop.EventLoop
}

func newNativeKVProxy(backing KV, vm *goja.Runtime, eventLoop *eventloop.EventLoop) *NativeKVProxy {
	p := &NativeKVProxy{
		vm:        vm,
		backing:   backing,
		eventLoop: eventLoop,
	}
	p.nativeObj = vm.NewDynamicObject(p)
	return p
}

var kvProperties = []string{}

var _ goja.DynamicObject = (*NativeKVProxy)(nil)

func (kv *NativeKVProxy) get(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	key := fc.Argument(0).String()
	go func() {
		val, err := kv.backing.Get(kv.ioContext.Context(), key)
		kv.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			if err != nil {
				if errors.Is(err, ErrKeyNotFound) {
					resolve(goja.Null())
					return
				}
				reject(vm.NewGoError(err))
				return
			}
			resolve(string(val))
		})
	}()
	return vm.ToValue(promise)
}

func (kv *NativeKVProxy) put(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	key := fc.Argument(0).String()
	val := fc.Argument(1).String()
	go func() {
		err := kv.backing.Put(kv.ioContext.Context(), key, []byte(val))
		kv.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			if err != nil {
				reject(vm.NewGoError(err))
			} else {
				resolve(goja.Undefined())
			}
		})
	}()
	return vm.ToValue(promise)
}

func (kv *NativeKVProxy) del(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	key := fc.Argument(0).String()
	go func() {
		deleted, err := kv.backing.Del(kv.ioContext.Context(), key)
		kv.eventLoop.RunOnLoop(func(vm *goja.Runtime) {
			if err != nil {
				reject(vm.NewGoError(err))
			} else {
				resolve(deleted)
			}
		})
	}()
	return vm.ToValue(promise)
}

func (kv *NativeKVProxy) NativeObject() goja.Value {
	return kv.nativeObj
}

func (kv *NativeKVProxy) Get(key string) goja.Value {
	switch key {
	case "get":
		if kv.nativeGet == nil {
			kv.nativeGet = kv.vm.ToValue(kv.get)
		}
		return kv.nativeGet
	case "put":
		if kv.nativePut == nil {
			kv.nativePut = kv.vm.ToValue(kv.put)
		}
		return kv.nativePut
	case "del":
		if kv.nativeDel == nil {
			kv.nativeDel = kv.vm.ToValue(kv.del)
		}
		return kv.nativeDel
	default:
		return goja.Undefined()
	}
}

func (kv *NativeKVProxy) Set(key string, val goja.Value) bool {
	return false
}

func (kv *NativeKVProxy) Has(key string) bool {
	return false
}

func (kv *NativeKVProxy) Delete(key string) bool {
	return false
}

func (kv *NativeKVProxy) Keys() []string {
	return kvProperties
}

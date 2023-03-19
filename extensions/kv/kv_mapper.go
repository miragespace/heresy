package kv

import (
	"go.miragespace.co/heresy/extensions/common"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/puzpuzpuz/xsync/v2"
)

type KVMapper struct {
	vm            *goja.Runtime
	eventLoop     *eventloop.EventLoop
	kvProxyMap    map[string]*NativeKVProxy
	nativeObj     *goja.Object
	kvBackingKeys []string
}

var _ goja.DynamicObject = (*KVMapper)(nil)

func newKVMapper(bMap *xsync.MapOf[string, KV], vm *goja.Runtime, eventLoop *eventloop.EventLoop) *KVMapper {
	m := &KVMapper{
		vm:            vm,
		eventLoop:     eventLoop,
		kvProxyMap:    make(map[string]*NativeKVProxy, bMap.Size()),
		kvBackingKeys: make([]string, 0, bMap.Size()),
	}
	bMap.Range(func(key string, backing KV) bool {
		p := newNativeKVProxy(backing, m.vm, m.eventLoop)
		m.kvProxyMap[key] = p
		m.kvBackingKeys = append(m.kvBackingKeys, key)
		return true
	})
	m.nativeObj = vm.NewDynamicObject(m)
	return m
}

func (m *KVMapper) NativeObject() goja.Value {
	return m.nativeObj
}

func (m *KVMapper) WithIOContext(t *common.IOContext) {
	for k := range m.kvProxyMap {
		m.kvProxyMap[k].ioContext = t
	}
}

func (m *KVMapper) Reset() {
	for k := range m.kvProxyMap {
		m.kvProxyMap[k].ioContext = nil
	}
}

func (m *KVMapper) Get(key string) goja.Value {
	if m.kvProxyMap[key] != nil {
		return m.kvProxyMap[key].nativeObj
	}
	return goja.Undefined()
}

func (m *KVMapper) Set(key string, val goja.Value) bool {
	return false
}

func (m *KVMapper) Has(key string) bool {
	for _, k := range m.kvBackingKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (m *KVMapper) Delete(key string) bool {
	return false
}

func (m *KVMapper) Keys() []string {
	return m.kvBackingKeys
}

package kv

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/puzpuzpuz/xsync/v2"
)

type KVManager struct {
	kvMapping *xsync.MapOf[string, KV]
}

func NewKVManager() *KVManager {
	return &KVManager{
		kvMapping: xsync.NewMapOf[KV](),
	}
}

func (m *KVManager) Configure(name string, uri string) error {
	var (
		backing KV
		err     error
	)

	for _, s := range kvBacking {
		if !s.matcher(uri) {
			continue
		}

		backing, err = s.constructor(uri)
		if err != nil {
			return err
		}

		m.kvMapping.Store(name, backing)
		return nil
	}

	return ErrBackingNotFound
}

func (m *KVManager) GetKVMapper(vm *goja.Runtime, eventLoop *eventloop.EventLoop) *KVMapper {
	return newKVMapper(m.kvMapping, vm, eventLoop)
}

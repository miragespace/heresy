package memory

import (
	"context"
	"strings"

	"go.miragespace.co/heresy/extensions/kv"

	"github.com/puzpuzpuz/xsync/v2"
)

type MemoryKV struct {
	store *xsync.MapOf[string, []byte]
}

var _ kv.KV = (*MemoryKV)(nil)

func init() {
	kv.Register(NewMemoryKV, func(uri string) bool {
		return strings.HasPrefix(uri, "memory")
	})
}

func NewMemoryKV(uri string) (kv.KV, error) {
	return &MemoryKV{
		store: xsync.NewMapOf[[]byte](),
	}, nil
}

func (m *MemoryKV) Get(ctx context.Context, key string) (val []byte, err error) {
	v, ok := m.store.Load(key)
	if !ok {
		return nil, kv.ErrKeyNotFound
	}

	return v, nil
}

func (m *MemoryKV) Put(ctx context.Context, key string, val []byte) (err error) {
	m.store.Store(key, val)
	return nil
}

func (m *MemoryKV) Del(ctx context.Context, key string) (deleted bool, err error) {
	_, deleted = m.store.LoadAndDelete(key)
	return
}

package kv

import (
	"context"
	"fmt"
)

var (
	ErrKeyNotFound     = fmt.Errorf("kv: not found")
	ErrBackingNotFound = fmt.Errorf("kv: backing not found")
)

type KV interface {
	Get(ctx context.Context, key string) (val []byte, err error)
	Put(ctx context.Context, key string, val []byte) (err error)
	Del(ctx context.Context, key string) (deleted bool, err error)
}

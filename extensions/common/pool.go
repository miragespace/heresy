package common

import (
	"go.miragespace.co/heresy/extensions/common/shared"

	pool "github.com/libp2p/go-buffer-pool"
)

func GetBuffer() []byte {
	return pool.Get(shared.BufferSize)
}

func PutBuffer(buf []byte) {
	pool.Put(buf)
}

package common

import (
	pool "github.com/libp2p/go-buffer-pool"

	"go.miragespace.co/heresy/extensions/common/shared"
)

func GetBuffer() []byte {
	return pool.Get(shared.BufferSize)
}

func PutBuffer(buf []byte) {
	pool.Put(buf)
}

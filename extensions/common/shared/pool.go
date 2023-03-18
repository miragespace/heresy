package shared

import (
	pool "github.com/libp2p/go-buffer-pool"
)

func GetBuffer() []byte {
	return pool.Get(BufferSize)
}

func PutBuffer(buf []byte) {
	pool.Put(buf)
}

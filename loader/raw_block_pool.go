package loader

import "sync"

// RawBlockPool reuses large block buffers to avoid per-block heap allocation.
var RawBlockPool = sync.Pool{}

func GetRawBlockBuffer(size int) []byte {
	if pooled, ok := RawBlockPool.Get().([]byte); ok && cap(pooled) >= size {
		return pooled[:size]
	}
	return make([]byte, size)
}

func PutRawBlock(buf []byte) {
	if buf != nil {
		RawBlockPool.Put(buf[:0])
	}
}

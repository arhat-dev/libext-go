package util

import "sync"

var bytesBufPool = &sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32)
		return &buf
	},
}

func GetBytesBuf(size int) []byte {
	buf := *bytesBufPool.Get().(*([]byte))
	if len(buf) >= size {
		return buf
	}

	// resize to expected size
	extend := size - len(buf)
	if extend < size {
		return append(buf, make([]byte, extend)...)
	}
	return append(make([]byte, extend), buf...)
}

func PutBytesBuf(b *([]byte)) {
	bytesBufPool.Put(b)
}

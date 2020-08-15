package common

import (
	"bytes"
	"errors"
	"math/bits"
	"sync"
)

func init() {
	bufPools = InitBufPools()
	writeBufPool = InitWriteBufPool()
}

//
// Read buffer
//

var bufPools []sync.Pool

func InitBufPools() []sync.Pool {
	pools := make([]sync.Pool, 17) // 1B -> 64K
	for k := range pools {
		i := k
		pools[k].New = func() interface{} {
			return make([]byte, 1<<uint32(i))
		}
	}
	return pools
}

func msb(size int) uint16 {
	return uint16(bits.Len32(uint32(size)) - 1)
}

func GetBuffer(size int) []byte {
	if size <= 0 || size > 65536 {
		return nil
	}
	bits := msb(size)
	if size == 1<<bits {
		return bufPools[bits].Get().([]byte)[:size]
	}
	return bufPools[bits+1].Get().([]byte)[:size]
}

func PutBuffer(buf []byte) error {
	bits := msb(cap(buf))
	if cap(buf) == 0 || cap(buf) > 65536 || cap(buf) != 1<<bits {
		return errors.New("incorrect buffer size")
	}
	bufPools[bits].Put(buf)
	return nil
}

//
// Write buffer
//

var writeBufPool sync.Pool

func InitWriteBufPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} { return &bytes.Buffer{} },
	}
}

func GetWriteBuffer() *bytes.Buffer {
	return writeBufPool.Get().(*bytes.Buffer)
}

func PutWriteBuffer(buf *bytes.Buffer) {
	buf.Reset()
	writeBufPool.Put(buf)
}

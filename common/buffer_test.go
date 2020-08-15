package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	bufPools = InitBufPools()

	assert.Nil(t, GetBuffer(0))
	assert.Equal(t, 1, len(GetBuffer(1)))
	assert.Equal(t, 2, len(GetBuffer(2)))
	assert.Equal(t, 3, len(GetBuffer(3)))
	assert.Equal(t, 4, cap(GetBuffer(3)))
	assert.Equal(t, 4, cap(GetBuffer(4)))
	assert.Equal(t, 1023, len(GetBuffer(1023)))
	assert.Equal(t, 1024, cap(GetBuffer(1023)))
	assert.Equal(t, 1024, len(GetBuffer(1024)))
	assert.Equal(t, 65536, len(GetBuffer(65536)))
	assert.Nil(t, GetBuffer(65537))

	assert.NotNil(t, PutBuffer(nil), "put nil misbehavior")
	assert.NotNil(t, PutBuffer(make([]byte, 3, 3)), "put elem:3 []bytes misbehavior")
	assert.Nil(t, PutBuffer(make([]byte, 4, 4)), "put elem:4 []bytes misbehavior")
	assert.Nil(t, PutBuffer(make([]byte, 1023, 1024)), "put elem:1024 []bytes misbehavior")
	assert.Nil(t, PutBuffer(make([]byte, 65536, 65536)), "put elem:65536 []bytes misbehavior")
	assert.NotNil(t, PutBuffer(make([]byte, 65537, 65537)), "put elem:65537 []bytes misbehavior")

	data := GetBuffer(4)
	PutBuffer(data)
	newData := GetBuffer(4)
	assert.Equal(t, cap(data), cap(newData), "different cap while alloc.Get()")
}

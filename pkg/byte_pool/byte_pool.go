package byte_pool

import "backup/consts"

var DefaultBytePool = NewBytePool(consts.Size4MB, 100)

type BytePool struct {
	buffer chan []byte
	size   int
}

func NewBytePool(size int, cap int) *BytePool {
	pool := &BytePool{
		buffer: make(chan []byte, cap),
		size:   size,
	}
	for i := 0; i < cap; i++ {
		pool.buffer <- make([]byte, size)
	}
	return pool
}

func (p *BytePool) Get() []byte {
	select {
	case buf := <-p.buffer:
		buf = buf[:cap(buf)]
		return buf
	}
}

func (p *BytePool) Put(buf []byte) {
	select {
	case p.buffer <- buf:
	}
}

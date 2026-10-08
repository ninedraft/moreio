package moreio

import (
	"io"
	"sync/atomic"
)

// CounterWriter stores atomic counter of written bytes.
type CounterWriter struct {
	counter atomic.Int64
	wr      io.Writer
}

// CountWrites records number of bytes written to dst.
func CountWrites(dst io.Writer) *CounterWriter {
	if dst == nil {
		panic("nil writer")
	}
	return &CounterWriter{
		wr: dst,
	}
}

// Written returns number of bytes written to dst writer.
// It's safe to call it concurrently with .Write method.
func (cnt *CounterWriter) WrittenBytes() int64 {
	return cnt.counter.Load()
}

// Write implements io.Writer interface.
func (cnt *CounterWriter) Write(data []byte) (int, error) {
	n, err := cnt.wr.Write(data)

	cnt.counter.Add(int64(n))

	return n, err
}

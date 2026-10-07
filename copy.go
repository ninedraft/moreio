package moreio

import (
	"io"
)

const defaultCopyBufferSize = 32 * 1024

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyMakeBuf(dst io.Writer, src io.Reader, buf *[]byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	if buf == nil {
		panic("empty shared buf ptr")
	}

	if cap(*buf) == 0 {
		*buf = make([]byte, defaultCopyBufferSize)
	}

	return io.CopyBuffer(dst, src, *buf)
}

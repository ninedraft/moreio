package moreio

import (
	"errors"
	"io"
)

const defaultCopyBufferSize = 32 * 1024

func makeDefaultCopyBuffer() []byte {
	return make([]byte, defaultCopyBufferSize)
}

var (
	errInvalidWrite = errors.New("moreio: invalid write result")
)

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyMakeBuf(dst io.Writer, src io.Reader, makeBuf func() []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rf, ok := dst.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	if makeBuf == nil {
		makeBuf = makeDefaultCopyBuffer
	}

	buf := makeBuf()

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func onceValue[T any](from func() T) func() T {
	var value T
	var done bool

	return func() T {
		if !done {
			done = true
			value = from()
		}

		return value
	}
}

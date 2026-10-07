// Package moreio joins readers, strings, and byte slices into an io.Writer.
// It also provides wrappers that limit bytes returned by a reader or accepted by a writer.
package moreio

import (
	"errors"
	"io"
	"math"
)

// ErrTooLarge reports an attempt to read or write beyond a byte limit.
var ErrTooLarge = errors.New("size limit exceeded")

// LimitReader returns a reader that yields at most n bytes from source.
// It returns ErrTooLarge when it detects data beyond the limit.
// Calls to Read consume at most n+1 bytes from source to detect excess data.
//
// A negative n or n == math.MaxInt64 disables the limit.
//
// If source implements io.WriterTo, the returned reader also implements io.WriterTo.
// A call to WriteTo can consume more than n+1 bytes from source.
func LimitReader(source io.Reader, n int64) io.Reader {
	if n == math.MaxInt64 || n < 0 {
		// limit overflow or unlimited
		return source
	}

	re := limitedReader{Source: source, N: n + 1}

	if wrTo, _ := source.(io.WriterTo); wrTo != nil {
		return &limitedReaderWriterTo{limitedReader: re, wr: wrTo}
	}

	return &re
}

type limitedReader struct {
	Source  io.Reader // underlying reader
	N       int64     // max bytes remaining
	LastErr error     // last error encountered
}

func (re *limitedReader) Read(p []byte) (int, error) {
	if re.N <= 0 || re.LastErr != nil {
		return 0, re.LastErr
	}

	if int64(len(p)) > re.N {
		p = p[0:re.N]
	}

	n, err := re.Source.Read(p)
	re.N -= int64(n)

	if re.N <= 0 { // hard limit reached
		if errors.Is(err, io.EOF) {
			err = nil
		}

		err = errors.Join(err, ErrTooLarge)
		n--
	}

	re.LastErr = err

	return n, re.LastErr
}

type limitedReaderWriterTo struct {
	limitedReader
	wr io.WriterTo
}

func (re *limitedReaderWriterTo) WriteTo(w io.Writer) (int64, error) {
	if re.N <= 0 || re.LastErr != nil {
		return 0, re.LastErr
	}

	lw := &limitedWriter{Dst: w, N: max(re.N-1, 0)}

	n, err := re.wr.WriteTo(lw)
	if err != nil {
		re.LastErr = err
	}
	re.N -= n

	return n, re.LastErr
}

// LimitWriter returns a writer that accepts at most n bytes.
// It returns ErrTooLarge when it detects input beyond the limit.
//
// A negative n or n == math.MaxInt64 disables the limit.
//
// If dst implements io.ReaderFrom, the returned writer also implements io.ReaderFrom.
func LimitWriter(dst io.Writer, n int64) io.Writer {
	if n == math.MaxInt64 || n < 0 {
		// limit overflow or unlimited
		return dst
	}

	wr := limitedWriter{Dst: dst, N: n}
	if reFrom, _ := dst.(io.ReaderFrom); reFrom != nil {
		return &limitedReaderFrom{limitedWriter: wr, rd: reFrom}
	}

	return &wr
}

type limitedWriter struct {
	Dst     io.Writer
	N       int64 // max bytes remaining to write
	LastErr error
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.LastErr != nil {
		return 0, lw.LastErr
	}

	if lw.N <= 0 {
		if len(p) == 0 {
			return 0, nil
		}
		lw.LastErr = ErrTooLarge
		return 0, lw.LastErr
	}

	if int64(len(p)) > lw.N {
		n, err := lw.Dst.Write(p[:lw.N])
		lw.N -= int64(n)
		lw.LastErr = errors.Join(err, ErrTooLarge)
		return n, lw.LastErr
	}

	n, err := lw.Dst.Write(p)
	lw.N -= int64(n)

	lw.LastErr = err

	return n, lw.LastErr
}

type limitedReaderFrom struct {
	limitedWriter
	rd io.ReaderFrom
}

func (re *limitedReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	if re.LastErr != nil {
		return 0, re.LastErr
	}

	lr := &limitedReader{Source: r, N: re.N + 1}

	n, err := re.rd.ReadFrom(lr)
	if err != nil {
		re.LastErr = err
	}

	re.N = max(lr.N-1, 0)

	return n, re.LastErr
}

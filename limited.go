package moreio

import (
	"errors"
	"io"
	"math"
)

var ErrTooLarge = errors.New("size limit exceeded")

func LimitedReader(source io.Reader, n int64) io.Reader {
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

func LimitedWriter(dst io.Writer, n int64) io.Writer {
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
		lw.LastErr = ErrTooLarge
		return 0, lw.LastErr
	}

	if int64(len(p)) > lw.N {
		n, err := lw.Dst.Write(p[:lw.N])
		lw.N -= int64(n)
		lw.LastErr = errors.Join(err, ErrTooLarge)
		return 0, lw.LastErr
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
	if re.N <= 0 || re.LastErr != nil {
		return 0, re.LastErr
	}

	lr := &limitedReader{Source: r, N: re.N + 1}

	n, err := re.rd.ReadFrom(lr)
	if err != nil {
		re.LastErr = err
	}

	remain := lr.N - 1
	if remain < 0 {
		remain = 0
	}
	re.N = remain

	return n, re.LastErr
}

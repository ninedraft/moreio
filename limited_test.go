package moreio_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/ninedraft/moreio"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type failOnceReader struct {
	src    io.Reader
	err    error
	firstN int
	calls  int
}

func (r *failOnceReader) Read(p []byte) (int, error) {
	r.calls++
	if r.calls == 1 {
		if r.firstN == 0 {
			return 0, r.err
		}
		n, _ := r.src.Read(p[:min(len(p), r.firstN)])
		return n, r.err
	}
	return r.src.Read(p)
}

type failOnceWriter struct {
	dst    io.Writer
	err    error
	firstN int
	calls  int
}

func (w *failOnceWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 1 {
		if w.firstN == 0 {
			return 0, w.err
		}
		n, _ := w.dst.Write(p[:min(len(p), w.firstN)])
		return n, w.err
	}
	return w.dst.Write(p)
}

func TestLimitReaderRetriesAfterError(t *testing.T) {
	readErr := errors.New("read failed")
	src := &failOnceReader{src: strings.NewReader("ab"), err: readErr, firstN: 1}
	r := moreio.LimitReader(src, 2)
	buf := make([]byte, 2)

	n, err := r.Read(buf)
	if n != 1 || !errors.Is(err, readErr) || buf[0] != 'a' {
		t.Fatalf("first Read = (%d, %v), first byte = %q; want (1, %v), %q", n, err, buf[:1], readErr, "a")
	}
	n, err = r.Read(buf)
	if n != 1 || err != nil || buf[0] != 'b' || src.calls != 2 {
		t.Fatalf("retry Read = (%d, %v), first byte = %q, source calls = %d; want (1, nil), %q, 2 calls", n, err, buf[:1], src.calls, "b")
	}
}

func TestLimitWriterRetriesAfterError(t *testing.T) {
	writeErr := errors.New("write failed")
	var dst bytes.Buffer
	underlying := &failOnceWriter{dst: &dst, err: writeErr, firstN: 1}
	w := moreio.LimitWriter(underlying, 2)

	n, err := w.Write([]byte("ab"))
	if n != 1 || !errors.Is(err, writeErr) {
		t.Fatalf("first Write = (%d, %v), want (1, write failed)", n, err)
	}
	n, err = w.Write([]byte("b"))
	if n != 1 || err != nil || dst.String() != "ab" || underlying.calls != 2 {
		t.Fatalf("retry Write = (%d, %v), destination = %q, calls = %d; want (1, nil), %q, 2 calls", n, err, dst.String(), underlying.calls, "ab")
	}
}

func TestLimitReaderWriteToRetriesAfterError(t *testing.T) {
	writeErr := errors.New("write failed")
	var dst bytes.Buffer
	underlying := &failOnceWriter{dst: &dst, err: writeErr}
	wt := moreio.LimitReader(strings.NewReader("ab"), 2).(io.WriterTo)

	n, err := wt.WriteTo(underlying)
	if n != 0 || !errors.Is(err, writeErr) {
		t.Fatalf("first WriteTo = (%d, %v), want (0, write failed)", n, err)
	}
	n, err = wt.WriteTo(underlying)
	if n != 2 || err != nil || dst.String() != "ab" || underlying.calls != 2 {
		t.Fatalf("retry WriteTo = (%d, %v), destination = %q, calls = %d; want (2, nil), %q, 2 calls", n, err, dst.String(), underlying.calls, "ab")
	}
}

func TestLimitWriterReadFromRetriesAfterError(t *testing.T) {
	readErr := errors.New("read failed")
	var dst bytes.Buffer
	src := &failOnceReader{src: strings.NewReader("ab"), err: readErr}
	rf := moreio.LimitWriter(&dst, 2).(io.ReaderFrom)

	n, err := rf.ReadFrom(src)
	if n != 0 || !errors.Is(err, readErr) {
		t.Fatalf("first ReadFrom = (%d, %v), want (0, read failed)", n, err)
	}
	n, err = rf.ReadFrom(src)
	if n != 2 || err != nil || dst.String() != "ab" || src.calls < 2 {
		t.Fatalf("retry ReadFrom = (%d, %v), destination = %q, source calls = %d; want (2, nil), %q, at least 2 calls", n, err, dst.String(), src.calls, "ab")
	}
}

func TestUnderlyingErrTooLargeDoesNotExceedLimit(t *testing.T) {
	src := &failOnceReader{src: strings.NewReader("a"), err: moreio.ErrTooLarge}
	r := moreio.LimitReader(src, 1)
	buf := make([]byte, 1)
	if n, err := r.Read(buf); n != 0 || !errors.Is(err, moreio.ErrTooLarge) {
		t.Fatalf("first Read = (%d, %v), want (0, ErrTooLarge)", n, err)
	}
	if n, err := r.Read(buf); n != 1 || err != nil || buf[0] != 'a' {
		t.Fatalf("retry Read = (%d, %v), first byte = %q; want (1, nil), %q", n, err, buf[:1], "a")
	}

	var dst bytes.Buffer
	underlying := &failOnceWriter{dst: &dst, err: moreio.ErrTooLarge}
	w := moreio.LimitWriter(underlying, 1)
	if n, err := w.Write([]byte("a")); n != 0 || !errors.Is(err, moreio.ErrTooLarge) {
		t.Fatalf("first Write = (%d, %v), want (0, ErrTooLarge)", n, err)
	}
	if n, err := w.Write([]byte("a")); n != 1 || err != nil || dst.String() != "a" {
		t.Fatalf("retry Write = (%d, %v), destination = %q; want (1, nil), %q", n, err, dst.String(), "a")
	}
}

func TestLimitOverflowDoesNotReplayUnderlyingError(t *testing.T) {
	underlyingErr := errors.New("underlying failed")
	src := &failOnceReader{src: strings.NewReader("ab"), err: underlyingErr, firstN: 2}
	r := moreio.LimitReader(src, 1)
	buf := make([]byte, 2)
	n, err := r.Read(buf)
	if n != 1 || !errors.Is(err, underlyingErr) || !errors.Is(err, moreio.ErrTooLarge) {
		t.Fatalf("first Read = (%d, %v), want 1 byte with %v and ErrTooLarge", n, err, underlyingErr)
	}
	n, err = r.Read(buf)
	if n != 0 || !errors.Is(err, moreio.ErrTooLarge) || errors.Is(err, underlyingErr) || src.calls != 1 {
		t.Fatalf("retry Read = (%d, %v), source calls = %d; want (0, ErrTooLarge), 1 call", n, err, src.calls)
	}

	dst := &failOnceWriter{dst: io.Discard, err: underlyingErr}
	w := moreio.LimitWriter(dst, 1)
	n, err = w.Write([]byte("ab"))
	if n != 0 || !errors.Is(err, underlyingErr) || !errors.Is(err, moreio.ErrTooLarge) {
		t.Fatalf("first Write = (%d, %v), want 0 bytes with %v and ErrTooLarge", n, err, underlyingErr)
	}
	n, err = w.Write([]byte("a"))
	if n != 0 || !errors.Is(err, moreio.ErrTooLarge) || errors.Is(err, underlyingErr) || dst.calls != 1 {
		t.Fatalf("retry Write = (%d, %v), destination calls = %d; want (0, ErrTooLarge), 1 call", n, err, dst.calls)
	}
}

func TestLimitReaderRead(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		limit     int64
		want      string
		wantErr   error
		remaining int
	}{
		{name: "below limit", input: "ab", limit: 3, want: "ab"},
		{name: "at limit", input: "abc", limit: 3, want: "abc"},
		{name: "above limit", input: "abcdef", limit: 2, want: "ab", wantErr: moreio.ErrTooLarge, remaining: 3},
		{name: "zero limit and empty input", limit: 0},
		{name: "zero limit and nonempty input", input: "abc", limit: 0, wantErr: moreio.ErrTooLarge, remaining: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := strings.NewReader(tt.input)
			got, err := io.ReadAll(moreio.LimitReader(src, tt.limit))
			assert.Equal(t, tt.want, string(got))
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.remaining, src.Len())
		})
	}
}

func TestLimitReaderReadSticksAfterExcess(t *testing.T) {
	src := strings.NewReader("abcd")
	r := moreio.LimitReader(src, 2)
	buf := make([]byte, 4)

	n, err := r.Read(buf)
	assert.Equal(t, 2, n)
	assert.Equal(t, "ab", string(buf[:n]))
	assert.ErrorIs(t, err, moreio.ErrTooLarge)

	n, err = r.Read(buf)
	assert.Zero(t, n)
	assert.ErrorIs(t, err, moreio.ErrTooLarge)
	assert.Equal(t, 1, src.Len())
}

func TestLimitReaderUnlimited(t *testing.T) {
	for _, limit := range []int64{-1, math.MaxInt64} {
		src := strings.NewReader("abc")
		assert.Same(t, src, moreio.LimitReader(src, limit))
	}
}

func TestLimitReaderWithoutWriterTo(t *testing.T) {
	src := struct{ io.Reader }{Reader: strings.NewReader("ab")}
	r := moreio.LimitReader(src, 2)
	_, ok := r.(io.WriterTo)
	assert.False(t, ok)
	got, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "ab", string(got))
}

func TestLimitReaderWriterTo(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		limit   int64
		want    string
		wantErr error
	}{
		{name: "below limit", input: "ab", limit: 3, want: "ab"},
		{name: "at limit", input: "abc", limit: 3, want: "abc"},
		{name: "above limit", input: "abc", limit: 2, want: "ab", wantErr: moreio.ErrTooLarge},
		{name: "empty at zero limit", limit: 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			src := moreio.LimitReader(strings.NewReader(tt.input), tt.limit)
			n, err := io.Copy(&dst, src)
			assert.Equal(t, int64(len(tt.want)), n)
			assert.Equal(t, tt.want, dst.String())
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				n, err = io.Copy(&dst, src)
				assert.Zero(t, n)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.want, dst.String())
			}
		})
	}

	t.Run("destination error", func(t *testing.T) {
		wantErr := errors.New("destination failed")
		n, err := io.Copy(errorWriter{wantErr}, moreio.LimitReader(strings.NewReader("abc"), 2))
		assert.Zero(t, n)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestLimitWriterWrite(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		limit   int64
		want    string
		wantErr error
	}{
		{name: "below limit", input: "ab", limit: 3, want: "ab"},
		{name: "at limit", input: "abc", limit: 3, want: "abc"},
		{name: "above limit", input: "abc", limit: 2, want: "ab", wantErr: moreio.ErrTooLarge},
		{name: "empty at zero limit", limit: 0},
		{name: "nonempty at zero limit", input: "a", limit: 0, wantErr: moreio.ErrTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			n, err := moreio.LimitWriter(&dst, tt.limit).Write([]byte(tt.input))
			assert.Equal(t, len(tt.want), n)
			assert.Equal(t, tt.want, dst.String())
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLimitWriterWriteSticksAfterExcess(t *testing.T) {
	var dst bytes.Buffer
	w := moreio.LimitWriter(&dst, 2)
	_, err := w.Write([]byte("abc"))
	require.ErrorIs(t, err, moreio.ErrTooLarge)

	n, err := w.Write([]byte("d"))
	assert.Zero(t, n)
	assert.ErrorIs(t, err, moreio.ErrTooLarge)
	assert.Equal(t, "ab", dst.String())
}

func TestLimitWriterUnderlyingError(t *testing.T) {
	wantErr := errors.New("destination failed")
	for _, tt := range []struct {
		name         string
		input        string
		limit        int64
		wantTooLarge bool
	}{
		{name: "below limit", input: "ab", limit: 3},
		{name: "above limit", input: "abc", limit: 2, wantTooLarge: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			n, err := moreio.LimitWriter(errorWriter{wantErr}, tt.limit).Write([]byte(tt.input))
			assert.Zero(t, n)
			assert.ErrorIs(t, err, wantErr)
			assert.Equal(t, tt.wantTooLarge, errors.Is(err, moreio.ErrTooLarge))
		})
	}
}

func TestLimitWriterUnlimited(t *testing.T) {
	for _, limit := range []int64{-1, math.MaxInt64} {
		var dst bytes.Buffer
		assert.Same(t, &dst, moreio.LimitWriter(&dst, limit))
	}
}

func TestLimitWriterReadFromRejectsInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom, ok := moreio.LimitWriter(&dst, 0).(io.ReaderFrom)
	if !ok {
		t.Fatal("LimitWriter must preserve ReaderFrom")
	}
	if n, err := readFrom.ReadFrom(strings.NewReader("x")); n != 0 || !errors.Is(err, moreio.ErrTooLarge) || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, ErrTooLarge), empty destination", n, err, dst.String())
	}
}

func TestLimitWriterReadFromAllowsEmptyInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom := moreio.LimitWriter(&dst, 0).(io.ReaderFrom)
	if n, err := readFrom.ReadFrom(strings.NewReader("")); n != 0 || err != nil || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, nil), empty destination", n, err, dst.String())
	}
}

func TestLimitWriterReadFrom(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		limit   int64
		want    string
		wantErr error
	}{
		{name: "below limit", input: "ab", limit: 3, want: "ab"},
		{name: "at limit", input: "abc", limit: 3, want: "abc"},
		{name: "above limit", input: "abcd", limit: 2, want: "ab", wantErr: moreio.ErrTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			rf, ok := moreio.LimitWriter(&dst, tt.limit).(io.ReaderFrom)
			require.True(t, ok)
			n, err := rf.ReadFrom(strings.NewReader(tt.input))
			assert.Equal(t, int64(len(tt.want)), n)
			assert.Equal(t, tt.want, dst.String())
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				n, err = rf.ReadFrom(strings.NewReader("later"))
				assert.Zero(t, n)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.want, dst.String())
			}
		})
	}

	t.Run("source error", func(t *testing.T) {
		wantErr := errors.New("source failed")
		var dst bytes.Buffer
		rf := moreio.LimitWriter(&dst, 2).(io.ReaderFrom)
		n, err := rf.ReadFrom(iotest.ErrReader(wantErr))
		assert.Zero(t, n)
		assert.ErrorIs(t, err, wantErr)
	})
}

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

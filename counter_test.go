package moreio_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/ninedraft/moreio"
	"github.com/stretchr/testify/require"
)

func ExampleCountWrites() {
	b := &strings.Builder{}
	wr := moreio.CountWrites(b)

	n, _ := io.WriteString(wr, "Alice |> Sola]")

	fmt.Printf("%s %d=%d bytes\n", b, wr.WrittenBytes(), n)
	// Output: Alice |> Sola] 14=14 bytes
}

func TestCounterWrites(t *testing.T) {
	t.Parallel()

	t.Run("successive writes", func(t *testing.T) {
		var dst bytes.Buffer
		wr := moreio.CountWrites(&dst)
		require.Zero(t, wr.WrittenBytes())

		for _, tt := range []struct {
			name  string
			data  []byte
			want  string
			count int64
		}{
			{name: "multibyte", data: []byte("aλ"), want: "aλ", count: 3},
			{name: "empty", want: "aλ", count: 3},
			{name: "later write", data: []byte("b"), want: "aλb", count: 4},
		} {
			t.Run(tt.name, func(t *testing.T) {
				n, err := wr.Write(tt.data)
				require.NoError(t, err)
				require.Equal(t, len(tt.data), n)
				require.Equal(t, tt.want, dst.String())
				require.Equal(t, tt.count, wr.WrittenBytes())
			})
		}
	})

	t.Run("zero-progress error", func(t *testing.T) {
		wantErr := errors.New("write failed")
		wr := moreio.CountWrites(errorWriter{wantErr})
		n, err := wr.Write([]byte("abc"))
		require.Zero(t, n)
		require.ErrorIs(t, err, wantErr)
		require.Zero(t, wr.WrittenBytes())
	})

	t.Run("partial error then recovery", func(t *testing.T) {
		wantErr := errors.New("write failed")
		dst := &failingWriter{failAt: 1, n: 2, err: wantErr}
		wr := moreio.CountWrites(dst)

		n, err := wr.Write([]byte("abc"))
		require.Equal(t, 2, n)
		require.ErrorIs(t, err, wantErr)
		require.Equal(t, "ab", dst.String())
		require.Equal(t, int64(2), wr.WrittenBytes())

		n, err = wr.Write([]byte("de"))
		require.Equal(t, 2, n)
		require.NoError(t, err)
		require.Equal(t, "abde", dst.String())
		require.Equal(t, int64(4), wr.WrittenBytes())
	})

	t.Run("nil destination", func(t *testing.T) {
		require.Panics(t, func() { moreio.CountWrites(nil) })
	})
}

func ExampleCountReads() {
	data := strings.NewReader("sample text")
	buf := make([]byte, 6)

	counter := moreio.CountReads(data)
	n, _ := counter.Read(buf)
	fmt.Printf("%s %d=%d bytes\n", buf, counter.ReadBytes(), n)
	// Output: sample 6=6 bytes
}

func TestCountReads(t *testing.T) {
	t.Parallel()

	t.Run("successive reads", func(t *testing.T) {
		re := moreio.CountReads(iotest.OneByteReader(strings.NewReader("aλb")))
		require.Zero(t, re.ReadBytes())

		n, err := re.Read(nil)
		require.Zero(t, n)
		require.NoError(t, err)
		require.Zero(t, re.ReadBytes())

		buf := make([]byte, 2)
		for i, want := range []byte("aλb") {
			n, err := re.Read(buf)
			require.Equal(t, 1, n)
			require.NoError(t, err)
			require.Equal(t, want, buf[0])
			require.Equal(t, int64(i+1), re.ReadBytes())
		}

		n, err = re.Read(buf)
		require.Zero(t, n)
		require.ErrorIs(t, err, io.EOF)
		require.Equal(t, int64(len("aλb")), re.ReadBytes())
	})

	t.Run("zero-progress error", func(t *testing.T) {
		wantErr := errors.New("read failed")
		re := moreio.CountReads(iotest.ErrReader(wantErr))
		n, err := re.Read(make([]byte, 4))
		require.Zero(t, n)
		require.ErrorIs(t, err, wantErr)
		require.Zero(t, re.ReadBytes())
	})

	t.Run("data and error", func(t *testing.T) {
		wantErr := errors.New("read failed")
		src := iotest.DataErrReader(io.MultiReader(strings.NewReader("ab"), iotest.ErrReader(wantErr)))
		re := moreio.CountReads(src)
		buf := make([]byte, 4)
		n, err := re.Read(buf)
		require.Equal(t, 2, n)
		require.Equal(t, "ab", string(buf[:n]))
		require.ErrorIs(t, err, wantErr)
		require.Equal(t, int64(2), re.ReadBytes())
	})

	t.Run("transient error", func(t *testing.T) {
		re := moreio.CountReads(iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("ab"))))
		buf := make([]byte, 1)

		n, err := re.Read(buf)
		require.Equal(t, 1, n)
		require.NoError(t, err)
		require.Equal(t, byte('a'), buf[0])
		require.Equal(t, int64(1), re.ReadBytes())

		n, err = re.Read(buf)
		require.Zero(t, n)
		require.ErrorIs(t, err, iotest.ErrTimeout)
		require.Equal(t, int64(1), re.ReadBytes())

		n, err = re.Read(buf)
		require.Equal(t, 1, n)
		require.NoError(t, err)
		require.Equal(t, byte('b'), buf[0])
		require.Equal(t, int64(2), re.ReadBytes())
	})

	t.Run("read all", func(t *testing.T) {
		const sample = "sample data"
		re := moreio.CountReads(strings.NewReader(sample))
		got, err := io.ReadAll(re)
		require.NoError(t, err)
		require.Equal(t, sample, string(got))
		require.Equal(t, int64(len(sample)), re.ReadBytes())
	})

	t.Run("nil source", func(t *testing.T) {
		require.Panics(t, func() { moreio.CountReads(nil) })
	})
}

func TestCountersCopy(t *testing.T) {
	t.Parallel()

	const sample = "aλb"
	t.Run("writer", func(t *testing.T) {
		var dst bytes.Buffer
		wr := moreio.CountWrites(&dst)
		n, err := io.Copy(wr, strings.NewReader(sample))
		require.NoError(t, err)
		require.Equal(t, int64(len(sample)), n)
		require.Equal(t, sample, dst.String())
		require.Equal(t, n, wr.WrittenBytes())
	})

	t.Run("reader", func(t *testing.T) {
		var dst bytes.Buffer
		re := moreio.CountReads(strings.NewReader(sample))
		n, err := io.Copy(&dst, re)
		require.NoError(t, err)
		require.Equal(t, int64(len(sample)), n)
		require.Equal(t, sample, dst.String())
		require.Equal(t, n, re.ReadBytes())
	})
}

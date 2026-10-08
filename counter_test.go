package moreio_test

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/ninedraft/moreio"
	"github.com/stretchr/testify/require"
)

func ExampleCountWrites() {
	b := &strings.Builder{}
	wr := moreio.CountWrites(b)

	n, _ := io.WriteString(wr, "Alice |> Sola]")

	fmt.Printf("%s %d=%d bytes\n", b, wr.Written(), n)
	// Output: Alice |> Sola] 14=14 bytes
}

func TestCounterWrites(t *testing.T) {
	t.Parallel()
	sample := slices.Repeat([]byte{'1'}, 10)

	gotData := &bytes.Buffer{}

	wr := moreio.CountWrites(gotData)
	n, err := writeByByte(wr, sample)

	require.NoError(t, err, "moreio.CounterWriter.Write(sample).error")
	require.EqualValues(t, len(sample), n, "moreio.CounterWriter.Write(sample).n == len(sample)")
	require.Equal(t, int64(len(sample)), wr.Written(), "moreio.CounterWriter.Written() == len(sample)")
	require.Equal(t, string(sample), gotData.String(), "moreio.CounterWriter must pass bytes into dst unchanged")
}

func writeByByte(dst io.Writer, data []byte) (written int, _ error) {
	for i := range data {
		n, err := dst.Write(data[i : i+1])
		written += n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

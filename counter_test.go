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

	fmt.Printf("%s %d=%d bytes\n", b, wr.WrittenBytes(), n)
	// Output: Alice |> Sola] 14=14 bytes
}

func TestCounterWrites(t *testing.T) {
	t.Parallel()

	sample := slices.Repeat([]byte{'1'}, 10)

	gotData := &bytes.Buffer{}

	wr := moreio.CountWrites(gotData)
	n, err := wr.Write(sample)

	require.NoError(t, err, "moreio.CounterWriter.Write(sample).error")
	require.EqualValues(t, len(sample), n, "moreio.CounterWriter.Write(sample).n == len(sample)")
	require.Equal(t, int64(len(sample)), wr.WrittenBytes(), "moreio.CounterWriter.WrittenBytes() == len(sample)")
	require.Equal(t, string(sample), gotData.String(), "moreio.CounterWriter must pass bytes into dst unchanged")
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

	const sample = "sample data"

	re := moreio.CountReads(strings.NewReader(sample))

	got, err := io.ReadAll(re)

	require.NoError(t, err, "read all from moreio.CountReads(sample)")
	require.Equal(t, len(sample), len(got), "read all from moreio.CountReads(sample) bytes")
	require.Equal(t, int64(len(sample)), re.ReadBytes(), "moreio.CountReads(sample).ReadBytes")
}

package moreio_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ninedraft/moreio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinReadersCopyPaths(t *testing.T) {
	part := strings.Repeat("x", 64*1024+1)
	want := "abc|" + part

	t.Run("destination ReaderFrom", func(t *testing.T) {
		var dst bytes.Buffer
		first := struct{ io.Reader }{Reader: strings.NewReader("abc")}
		second := struct{ io.Reader }{Reader: strings.NewReader(part)}
		n, err := moreio.JoinReaders(&dst, []byte("|"), first, second)
		require.NoError(t, err)
		assert.Equal(t, int64(len(want)), n)
		assert.Equal(t, want, dst.String())
	})

	t.Run("plain Writer and Reader", func(t *testing.T) {
		var output bytes.Buffer
		dst := struct{ io.Writer }{Writer: &output}
		first := struct{ io.Reader }{Reader: strings.NewReader("abc")}
		second := struct{ io.Reader }{Reader: strings.NewReader(part)}
		n, err := moreio.JoinReaders(dst, []byte("|"), first, second)
		require.NoError(t, err)
		assert.Equal(t, int64(len(want)), n)
		assert.Equal(t, want, output.String())
	})
}

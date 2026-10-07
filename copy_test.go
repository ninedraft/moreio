package moreio

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCopyMakeBufZeroLengthWithCapacity(t *testing.T) {
	var output bytes.Buffer
	dst := struct{ io.Writer }{Writer: &output}
	src := struct{ io.Reader }{Reader: strings.NewReader("x")}
	buf := make([]byte, 0, 8)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("copyMakeBuf panicked with a buffer that has capacity: %v", r)
		}
	}()

	n, err := copyMakeBuf(dst, src, &buf)
	if n != 1 || err != nil || output.String() != "x" {
		t.Errorf("copyMakeBuf = (%d, %v), output = %q; want (1, nil), %q", n, err, output.String(), "x")
	}
}

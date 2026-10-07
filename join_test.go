package moreio_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ninedraft/moreio"
)

const lorem = `Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since 1966, when designers at Letraset and James Mosley, the librarian at St Bride Printing Library in London, took a 1914 Cicero translation and scrambled it to make dummy text for Letraset's Body Type sheets. It has survived not only many decades, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised thanks to these sheets and more recently with desktop publishing software like Aldus PageMaker and Microsoft Word including versions of Lorem Ipsum.`

var (
	loremStrings = strings.Fields(lorem)
	loremBytes   = bytes.Fields([]byte(lorem))
)

var discardWriterOnly = struct {
	io.Writer
}{Writer: io.Discard}

func BenchmarkJoinStrings_Writer(b *testing.B) {
	for b.Loop() {
		moreio.JoinStrings(discardWriterOnly, ", ", loremStrings...)
	}
}

func BenchmarkJoinStrings_StringWriter(b *testing.B) {
	for b.Loop() {
		moreio.JoinStrings(io.Discard, ", ", loremStrings...)
	}
}

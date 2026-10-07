package moreio_test

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"strings"
	"testing"

	"github.com/ninedraft/moreio"

	"github.com/stretchr/testify/assert"
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

func TestJoinReader(t *testing.T) {
	makeParts := func(parts []string) []io.Reader {
		pp := make([]io.Reader, 0, len(parts))
		for _, part := range parts {
			pp = append(pp, strings.NewReader(part))
		}
		return pp
	}

	tests := []struct {
		name  string
		sep   string
		parts []string
		want  int64
		err   error
	}{
		{
			name:  "base",
			parts: loremStrings,
			sep:   ", ",
			want:  731,
		},
		{
			name:  "empty separator",
			parts: loremStrings,
			want:  535,
		},
		{
			name:  "no parts",
			sep:   ", ",
			parts: nil,
			want:  0,
		},
		{
			name:  "single part",
			parts: []string{"only"},
			sep:   ", ",
			want:  4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &strings.Builder{}
			n, err := moreio.JoinReader(got, []byte(tt.sep), makeParts(tt.parts)...)

			assert.Equal(t, strings.Join(tt.parts, tt.sep), got.String(), "result must be equal to strings.Join")
			assert.Equal(t, tt.want, n, "moreio.JoinReader(sep, parts).written")
			assert.ErrorIs(t, err, tt.err, "moreio.JoinReader(sep, parts).err")
		})
	}
}

func TestJoinStrings(t *testing.T) {
	type buffer interface {
		io.Writer
		fmt.Stringer
	}

	tests := []struct {
		name string
		sep  string
		in   []string
		want string
	}{
		{name: "no parts", sep: ", ", want: ""},
		{name: "one part", sep: ", ", in: []string{"one"}, want: "one"},
		{name: "multiple parts", sep: ", ", in: []string{"one", "two", "three"}, want: "one, two, three"},
		{name: "empty separator", in: []string{"one", "two"}, want: "onetwo"},
		{name: "byte count", sep: "|", in: []string{"λ", "猫"}, want: "λ|猫"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, useStringWriter := range []bool{true, false} {
				var dst buffer = &bytes.Buffer{}
				if useStringWriter {
					dst = &strings.Builder{}
				}

				n, err := moreio.JoinStrings(dst, tt.sep, tt.in...)
				assert.NoError(t, err)
				assert.Equal(t, tt.want, dst.String())
				assert.Equal(t, int64(len(tt.want)), n)
			}
		})
	}
}

func TestJoinBytes(t *testing.T) {
	tests := []struct {
		name string
		sep  []byte
		in   [][]byte
		want string
	}{
		{name: "no parts", sep: []byte(", ")},
		{name: "one part", sep: []byte(", "), in: [][]byte{[]byte("one")}, want: "one"},
		{name: "multiple parts", sep: []byte(", "), in: [][]byte{[]byte("one"), []byte("two")}, want: "one, two"},
		{name: "empty separator", in: [][]byte{[]byte("one"), []byte("two")}, want: "onetwo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst bytes.Buffer
			n, err := moreio.JoinBytes(&dst, tt.sep, tt.in...)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, dst.String())
			assert.Equal(t, int64(len(tt.want)), n)
		})
	}
}

func TestJoinStream(t *testing.T) {
	tests := []struct {
		name  string
		sep   []byte
		parts []string
		want  string
	}{
		{name: "no parts", sep: []byte(", ")},
		{name: "one part", sep: []byte(", "), parts: []string{"one"}, want: "one"},
		{name: "multiple parts", sep: []byte(", "), parts: []string{"one", "two"}, want: "one, two"},
		{name: "empty separator", parts: []string{"one", "two"}, want: "onetwo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := func(yield func(io.Reader) bool) {
				for _, p := range tt.parts {
					if !yield(strings.NewReader(p)) {
						return
					}
				}
			}
			var dst bytes.Buffer
			n, err := moreio.JoinStream(&dst, tt.sep, iter.Seq[io.Reader](parts))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, dst.String())
			assert.Equal(t, int64(len(tt.want)), n)
		})
	}
}

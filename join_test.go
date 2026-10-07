package moreio_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
	"testing"

	"github.com/ninedraft/moreio"

	"github.com/stretchr/testify/assert"
)

type failingWriter struct {
	buf    bytes.Buffer
	calls  int
	failAt int
	n      int
	err    error
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		n, _ := w.buf.Write(p[:w.n])
		return n, w.err
	}
	return w.buf.Write(p)
}

func (w *failingWriter) String() string { return w.buf.String() }

const lorem = `Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since 1966, when designers at Letraset and James Mosley, the librarian at St Bride Printing Library in London, took a 1914 Cicero translation and scrambled it to make dummy text for Letraset's Body Type sheets. It has survived not only many decades, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised thanks to these sheets and more recently with desktop publishing software like Aldus PageMaker and Microsoft Word including versions of Lorem Ipsum.`

var loremStrings = strings.Fields(lorem)

var discardWriterOnly = struct {
	io.Writer
}{Writer: io.Discard}

func BenchmarkJoinStrings_Writer(b *testing.B) {
	for b.Loop() {
		if _, err := moreio.JoinStrings(discardWriterOnly, ", ", loremStrings...); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJoinStrings_StringWriter(b *testing.B) {
	for b.Loop() {
		if _, err := moreio.JoinStrings(io.Discard, ", ", loremStrings...); err != nil {
			b.Fatal(err)
		}
	}
}

func TestJoinReaders(t *testing.T) {
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
			n, err := moreio.JoinReaders(got, []byte(tt.sep), makeParts(tt.parts)...)

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

func TestJoinReadersSeq(t *testing.T) {
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
			n, err := moreio.JoinReadersSeq(&dst, tt.sep, iter.Seq[io.Reader](parts))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, dst.String())
			assert.Equal(t, int64(len(tt.want)), n)
		})
	}
}

func TestJoinFailures(t *testing.T) {
	parts := []string{"ab", "cd", "ef"}
	joiners := []struct {
		name string
		join func(io.Writer) (int64, error)
	}{
		{
			name: "readers",
			join: func(dst io.Writer) (int64, error) {
				return moreio.JoinReaders(dst, []byte("|"), strings.NewReader(parts[0]), strings.NewReader(parts[1]), strings.NewReader(parts[2]))
			},
		},
		{
			name: "reader sequence",
			join: func(dst io.Writer) (int64, error) {
				seq := func(yield func(io.Reader) bool) {
					for _, part := range parts {
						if !yield(strings.NewReader(part)) {
							return
						}
					}
				}
				return moreio.JoinReadersSeq(dst, []byte("|"), seq)
			},
		},
		{
			name: "strings through Writer",
			join: func(dst io.Writer) (int64, error) {
				return moreio.JoinStrings(dst, "|", parts...)
			},
		},
		{
			name: "bytes",
			join: func(dst io.Writer) (int64, error) {
				return moreio.JoinBytes(dst, []byte("|"), []byte(parts[0]), []byte(parts[1]), []byte(parts[2]))
			},
		},
	}

	wantErr := errors.New("write failed")
	cases := []struct {
		name    string
		failAt  int
		n       int
		err     error
		want    string
		wantErr error
	}{
		{name: "short first part", failAt: 1, n: 1, want: "a", wantErr: io.ErrShortWrite},
		{name: "short separator", failAt: 2, want: "ab", wantErr: io.ErrShortWrite},
		{name: "separator error", failAt: 2, err: wantErr, want: "ab", wantErr: wantErr},
		{name: "short later part", failAt: 3, n: 1, want: "ab|c", wantErr: io.ErrShortWrite},
		{name: "later part error", failAt: 3, n: 1, err: wantErr, want: "ab|c", wantErr: wantErr},
	}
	for _, joiner := range joiners {
		t.Run(joiner.name, func(t *testing.T) {
			for _, tt := range cases {
				t.Run(tt.name, func(t *testing.T) {
					dst := &failingWriter{failAt: tt.failAt, n: tt.n, err: tt.err}
					n, err := joiner.join(dst)
					assert.Equal(t, int64(len(tt.want)), n)
					assert.Equal(t, tt.want, dst.String())
					assert.ErrorIs(t, err, tt.wantErr)
					assert.Equal(t, tt.failAt, dst.calls)
				})
			}
		})
	}
}

func TestJoinReadersSeqStopsAfterError(t *testing.T) {
	wantErr := errors.New("write failed")
	dst := &failingWriter{failAt: 2, err: wantErr}
	yielded := 0
	seq := func(yield func(io.Reader) bool) {
		for _, part := range []string{"ab", "cd", "ef"} {
			yielded++
			if !yield(strings.NewReader(part)) {
				return
			}
		}
	}

	n, err := moreio.JoinReadersSeq(dst, []byte("|"), seq)
	assert.Equal(t, int64(2), n)
	assert.Equal(t, "ab", dst.String())
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 2, yielded)
}

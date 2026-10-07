package moreio_test

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ninedraft/moreio"
)

func ExampleJoinReader() {
	moreio.JoinReader(os.Stdout, []byte("|"),
		strings.NewReader("foo"),
		bytes.NewReader([]byte("bar")),
		bytes.NewBufferString("lat"))
	// Output: foo|bar|lat
}

func ExampleJoinBytes() {
	moreio.JoinBytes(os.Stdout, []byte("|"),
		[]byte("foo"),
		[]byte("bar"),
		[]byte("lat"))
	// Output: foo|bar|lat
}

func ExampleJoinStrings() {
	moreio.JoinStrings(os.Stdout, "|",
		"foo", "bar", "lat")
	// Output: foo|bar|lat
}

func ExampleJoinStream() {
	seq := func(yield func(io.Reader) bool) {
		for i := range 5 {
			re := strings.NewReader(strconv.Itoa(i))
			if !yield(re) {
				break
			}
		}
	}

	moreio.JoinStream(os.Stdout, []byte("|"), seq)
	// Output: 0|1|2|3|4
}

package moreio_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ninedraft/moreio"
)

func ExampleJoinReaders() {
	_, _ = moreio.JoinReaders(os.Stdout, []byte("|"),
		strings.NewReader("foo"),
		bytes.NewReader([]byte("bar")),
		bytes.NewBufferString("lat"))
	// Output: foo|bar|lat
}

func ExampleJoinReadersSeq() {
	seq := func(yield func(io.Reader) bool) {
		for i := range 5 {
			re := strings.NewReader(strconv.Itoa(i))
			if !yield(re) {
				break
			}
		}
	}

	_, _ = moreio.JoinReadersSeq(os.Stdout, []byte("|"), seq)
	// Output: 0|1|2|3|4
}

func ExampleJoinBytes() {
	_, _ = moreio.JoinBytes(os.Stdout, []byte("|"),
		[]byte("foo"),
		[]byte("bar"),
		[]byte("lat"))
	// Output: foo|bar|lat
}

func ExampleJoinStrings() {
	_, _ = moreio.JoinStrings(os.Stdout, "|",
		"foo", "bar", "lat")
	// Output: foo|bar|lat
}

func ExampleLimitReader() {
	var dst bytes.Buffer
	n, err := io.Copy(&dst, moreio.LimitReader(strings.NewReader("abcd"), 3))
	fmt.Println(dst.String(), n, errors.Is(err, moreio.ErrTooLarge))
	// Output: abc 3 true
}

func ExampleLimitWriter() {
	var dst bytes.Buffer
	n, err := moreio.LimitWriter(&dst, 3).Write([]byte("abcd"))
	fmt.Println(dst.String(), n, errors.Is(err, moreio.ErrTooLarge))
	// Output: abc 3 true
}

package moreio

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLimitedWriterReportsWrittenBytes(t *testing.T) {
	var dst bytes.Buffer
	n, err := LimitedWriter(&dst, 2).Write([]byte("abc"))
	if n != 2 || !errors.Is(err, ErrTooLarge) || dst.String() != "ab" {
		t.Errorf("Write = (%d, %v), destination = %q; want (2, ErrTooLarge), %q", n, err, dst.String(), "ab")
	}
}

func TestLimitedReaderWriterToReportsWrittenBytes(t *testing.T) {
	var dst bytes.Buffer
	n, err := io.Copy(&dst, LimitedReader(strings.NewReader("abc"), 2))
	if n != 2 || !errors.Is(err, ErrTooLarge) || dst.String() != "ab" {
		t.Errorf("Copy = (%d, %v), destination = %q; want (2, ErrTooLarge), %q", n, err, dst.String(), "ab")
	}
}

func TestLimitedWriterAllowsEmptyInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	w := LimitedWriter(&dst, 0)
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Errorf("empty Write = (%d, %v); want (0, nil)", n, err)
	}
}

func TestLimitedWriterReadFromRejectsInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom, ok := LimitedWriter(&dst, 0).(io.ReaderFrom)
	if !ok {
		t.Fatal("LimitedWriter must preserve ReaderFrom")
	}
	if n, err := readFrom.ReadFrom(strings.NewReader("x")); n != 0 || !errors.Is(err, ErrTooLarge) || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, ErrTooLarge), empty destination", n, err, dst.String())
	}
}

func TestLimitedWriterReadFromAllowsEmptyInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom := LimitedWriter(&dst, 0).(io.ReaderFrom)
	if n, err := readFrom.ReadFrom(strings.NewReader("")); n != 0 || err != nil || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, nil), empty destination", n, err, dst.String())
	}
}

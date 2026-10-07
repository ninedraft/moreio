package moreio

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLimitWriterReportsWrittenBytes(t *testing.T) {
	var dst bytes.Buffer
	n, err := LimitWriter(&dst, 2).Write([]byte("abc"))
	if n != 2 || !errors.Is(err, ErrTooLarge) || dst.String() != "ab" {
		t.Errorf("Write = (%d, %v), destination = %q; want (2, ErrTooLarge), %q", n, err, dst.String(), "ab")
	}
}

func TestLimitReaderWriterToReportsWrittenBytes(t *testing.T) {
	var dst bytes.Buffer
	n, err := io.Copy(&dst, LimitReader(strings.NewReader("abc"), 2))
	if n != 2 || !errors.Is(err, ErrTooLarge) || dst.String() != "ab" {
		t.Errorf("Copy = (%d, %v), destination = %q; want (2, ErrTooLarge), %q", n, err, dst.String(), "ab")
	}
}

func TestLimitWriterAllowsEmptyInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	w := LimitWriter(&dst, 0)
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Errorf("empty Write = (%d, %v); want (0, nil)", n, err)
	}
}

func TestLimitWriterReadFromRejectsInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom, ok := LimitWriter(&dst, 0).(io.ReaderFrom)
	if !ok {
		t.Fatal("LimitWriter must preserve ReaderFrom")
	}
	if n, err := readFrom.ReadFrom(strings.NewReader("x")); n != 0 || !errors.Is(err, ErrTooLarge) || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, ErrTooLarge), empty destination", n, err, dst.String())
	}
}

func TestLimitWriterReadFromAllowsEmptyInputAtLimit(t *testing.T) {
	var dst bytes.Buffer
	readFrom := LimitWriter(&dst, 0).(io.ReaderFrom)
	if n, err := readFrom.ReadFrom(strings.NewReader("")); n != 0 || err != nil || dst.Len() != 0 {
		t.Errorf("ReadFrom = (%d, %v), destination = %q; want (0, nil), empty destination", n, err, dst.String())
	}
}

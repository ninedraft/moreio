package moreio

import (
	"io"
	"iter"
)

// JoinReaders copies data from each reader to dst.
// It writes sep between readers.
// It returns the number of bytes written and the first error.
func JoinReaders(dst io.Writer, sep []byte, parts ...io.Reader) (written int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	var buf []byte

	if len(parts) > 0 {
		n, err := copyMakeBuf(dst, parts[0], &buf)
		written += n
		if err != nil {
			return written, err
		}
	}

	for _, part := range parts[1:] {
		if len(sep) > 0 {
			n, err := dst.Write(sep)
			written += int64(n)
			if err != nil {
				return written, err
			}
			if n != len(sep) {
				return written, io.ErrShortWrite
			}
		}

		n, err := copyMakeBuf(dst, part, &buf)
		written += n
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

// JoinReadersSeq copies data from each reader to dst.
// It writes sep between readers.
// It returns the number of bytes written and the first error.
func JoinReadersSeq(dst io.Writer, sep []byte, parts iter.Seq[io.Reader]) (written int64, err error) {
	first := true
	var buf []byte

	for part := range parts {
		if !first && len(sep) > 0 {
			n, err := dst.Write(sep)
			written += int64(n)
			if err != nil {
				return written, err
			}
			if n != len(sep) {
				return written, io.ErrShortWrite
			}
		}
		first = false

		n, err := copyMakeBuf(dst, part, &buf)
		written += n

		if err != nil {
			return written, err
		}
	}

	return written, nil
}

// JoinStrings writes each string to dst.
// It writes sep between strings.
// It returns the number of bytes written and the first error.
// It calls dst.WriteString if dst implements io.StringWriter.
func JoinStrings(dst io.Writer, sep string, parts ...string) (written int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	var buf []byte
	write := func(str string) (int, error) {
		buf = buf[:0]
		buf = append(buf, str...)

		return dst.Write(buf)
	}

	if strWriter, _ := dst.(io.StringWriter); strWriter != nil {
		write = strWriter.WriteString
	}

	if len(parts) > 0 {
		part := parts[0]

		n, err := write(part)
		written += int64(n)
		if err != nil {
			return written, err
		}

		if n != len(part) {
			return written, io.ErrShortWrite
		}
	}

	for _, part := range parts[1:] {
		if sep != "" {
			n, err := write(sep)
			written += int64(n)
			if err != nil {
				return written, err
			}
			if n != len(sep) {
				return written, io.ErrShortWrite
			}
		}

		n, err := write(part)
		written += int64(n)
		if err != nil {
			return written, err
		}
		if n != len(part) {
			return written, io.ErrShortWrite
		}
	}

	return written, nil
}

// JoinBytes writes each byte slice to dst.
// It writes sep between slices.
// It returns the number of bytes written and the first error.
func JoinBytes(dst io.Writer, sep []byte, parts ...[]byte) (written int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	for i, part := range parts {
		if i > 0 && len(sep) > 0 {
			n, err := dst.Write(sep)
			written += int64(n)
			if err != nil {
				return written, err
			}
			if n != len(sep) {
				return written, io.ErrShortWrite
			}
		}

		n, err := dst.Write(part)
		written += int64(n)
		if err != nil {
			return written, err
		}
		if n != len(part) {
			return written, io.ErrShortWrite
		}
	}

	return written, nil
}

package moreio

import (
	"io"
	"iter"
)

func JoinReader(dst io.Writer, sep []byte, parts ...io.Reader) (n int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	buf := onceValue(makeDefaultCopyBuffer)

	for i, p := range parts {
		if i > 0 && len(sep) > 0 {
			m, err := dst.Write(sep)
			n += int64(m)
			if err != nil {
				return n, err
			}
		}

		m, err := copyMakeBuf(dst, p, buf)
		n += m
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

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
		n, err := write(parts[0])
		written += int64(n)
		if err != nil {
			return written, err
		}
	}

	for _, p := range parts[1:] {
		n, err := write(sep)
		written += int64(n)
		if err != nil {
			return written, err
		}

		n, err = write(p)
		written += int64(n)
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

func JoinBytes(dst io.Writer, sep []byte, parts ...[]byte) (n int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	for i, p := range parts {
		if i > 0 && len(sep) > 0 {
			m, err := dst.Write(sep)
			n += int64(m)
			if err != nil {
				return n, err
			}
		}

		m, err := dst.Write(p)
		n += int64(m)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func JoinStream(dst io.Writer, sep []byte, parts iter.Seq[io.Reader]) (n int64, err error) {
	first := true
	buf := onceValue(makeDefaultCopyBuffer)

	for part := range parts {
		if !first && len(sep) > 0 {
			m, err := dst.Write(sep)
			n += int64(m)
			if err != nil {
				return n, err
			}
		}
		first = false

		m, err := copyMakeBuf(dst, part, buf)
		n += m

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

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

func JoinStrings(dst io.Writer, sep string, parts ...string) (n int64, err error) {
	if len(parts) == 0 {
		return 0, nil
	}

	var buf []byte
	write := func(str string) error {
		buf = buf[:0]
		buf = append(buf, str...)

		m, err := dst.Write(buf)
		n += int64(m)
		return err
	}

	if strWriter, _ := dst.(io.StringWriter); strWriter != nil {
		write = func(str string) error {
			m, err := strWriter.WriteString(str)
			n += int64(m)
			return err
		}
	}

	if len(parts) > 0 {
		if len(sep) > 0 {
			err := write(sep)
			if err != nil {
				return n, err
			}
		}

		err := write(parts[0])
		if err != nil {
			return n, err
		}
	}

	for i, p := range parts[1:] {
		if i > 0 && len(sep) > 0 {
			err := write(sep)
			if err != nil {
				return n, err
			}
		}

		err := write(p)
		if err != nil {
			return n, err
		}
	}

	return n, nil
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

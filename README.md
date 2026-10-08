[![Go Reference](https://pkg.go.dev/badge/github.com/ninedraft/moreio.svg)](https://pkg.go.dev/github.com/ninedraft/moreio)
[![Go](https://github.com/ninedraft/moreio/actions/workflows/go.yml/badge.svg)](https://github.com/ninedraft/moreio/actions/workflows/go.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/ninedraft/moreio)](https://github.com/ninedraft/moreio/blob/master/go.mod)
[![License](https://img.shields.io/github/license/ninedraft/moreio)](https://github.com/ninedraft/moreio/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/ninedraft/moreio)](https://github.com/ninedraft/moreio/releases)


# moreio

Collection of missing io streams utilities for golang.

Provides stricter limited writers and readers, more join functions, counters, etc.

## Import

```sh
go get github.com/ninedraft/moreio@latest
```

## Use


```go
import "github.com/ninedraft/moreio"

const messageMaxSize = 100 << 10

func handle(rw http.ResponseWriter, req *http.Request) {
	var message Message 
	err := json.UnmarshalRead(moreio.LimitReads(req.Body, messageMaxSize))
	
	if errors.Is(err, moreio.ErrTooLarge) {
		http.Error(rw, "request is too large", http.StatusRequestEntityTooLarge)
		return 
	}
	
	// ...
}
```

Other things available:
- `func JoinBytes(dst io.Writer, sep []byte, parts ...[]byte) (written int64, err error)`
- `func JoinReaders(dst io.Writer, sep []byte, parts ...io.Reader) (written int64, err error)`
- `func JoinReadersSeq(dst io.Writer, sep []byte, parts iter.Seq[io.Reader]) (written int64, err error)`
- `func JoinStrings(dst io.Writer, sep string, parts ...string) (written int64, err error)`
- `func LimitReader(source io.Reader, n int64) io.Reader`
- `func LimitWriter(dst io.Writer, n int64) io.Writer`
- `func CountReads(src io.Reader) *CounterReader`
- `func CountWrites(dst io.Writer) *CounterWriter`

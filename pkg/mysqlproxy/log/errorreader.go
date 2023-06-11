package log

import "io"

type errorReader struct {
	io.Reader
	err error
}

func NewErrorReader(r io.Reader) *errorReader { return &errorReader{Reader: r} }

func (e *errorReader) Read(p []byte) (n int, err error) {
	n, err = e.Reader.Read(p)
	if err != nil {
		e.err = err
	}
	return n, err
}
func (e *errorReader) Error() error { return e.err }

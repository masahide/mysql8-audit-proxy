package timeoutnet

import (
	"context"
	"io"
	"net"
	"time"
)

type TimeoutWriter struct {
	Conn    net.Conn
	Timeout time.Duration
	Ctx     context.Context
}

func (w *TimeoutWriter) Write(p []byte) (int, error) {
	select {
	case <-w.Ctx.Done():
		return 0, w.Ctx.Err()
	default:
	}
	w.Conn.SetWriteDeadline(time.Now().Add(w.Timeout))
	return w.Conn.Write(p)
}

// TimeoutReader wraps a net.Conn and sets a read timeout
type TimeoutReader struct {
	net.Conn
	Timeout time.Duration
	Ctx     context.Context
}

func (r *TimeoutReader) Read(p []byte) (int, error) {
	select {
	case <-r.Ctx.Done():
		return 0, r.Ctx.Err()
	default:
	}
	r.Conn.SetReadDeadline(time.Now().Add(r.Timeout))
	return r.Conn.Read(p)
}
func (r *TimeoutReader) WriteTo(w io.Writer) (int64, error) {
	var totalWritten int64
	buf := make([]byte, 1024)

	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				// Graceful connection close
				break
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Read operation timed out
				// Handle timeout and ncontinue reading with new deadline
				continue
			} else {
				// Other error
				return totalWritten, err
			}
		}

		written, err := w.Write(buf[:n])
		totalWritten += int64(written)
		if err != nil {
			return totalWritten, err
		}
	}

	return totalWritten, nil
}

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
	w.Conn.SetWriteDeadline(time.Now().Add(w.Timeout))
	n, err := w.Conn.Write(p)

	select {
	case <-w.Ctx.Done():
		// Context canceled or expired
		return n, w.Ctx.Err()
	default:
		return n, err
	}
}

// TimeoutReader wraps a net.Conn and sets a read timeout
type TimeoutReader struct {
	net.Conn
	Timeout time.Duration
	Ctx     context.Context
}

func (r *TimeoutReader) Read(p []byte) (int, error) {
	r.Conn.SetReadDeadline(time.Now().Add(r.Timeout))
	n, err := r.Conn.Read(p)

	select {
	case <-r.Ctx.Done():
		// Context canceled or expired
		return n, r.Ctx.Err()
	default:
		return n, err
	}
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
				// Handle timeout and continue reading with new deadline
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

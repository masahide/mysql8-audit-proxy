package mysqlproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
)

type LogWriter interface {
	PushToLogChannel(ctx context.Context, sp *sendpacket.SendPacket) error
	PutSendPacket(b *sendpacket.SendPacket)
	GetSendPacket() *sendpacket.SendPacket
	CloseChannel()
}

type SendTask struct {
	Reader net.Conn
	Writer io.Writer
	User   string
	DB     string
	Addr   string
	ConnID uint32
	Config *ProxyCfg
	LogWriter
}

func (st *SendTask) Worker(ctx context.Context) error {
	var sp *sendpacket.SendPacket
	defer func() {
		if sp != nil {
			st.PutSendPacket(sp)
		}
		st.sendState(ctx, "disconnect")
	}()
	st.sendState(ctx, "connect")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if sp == nil {
			sp = st.newSendPacket()
		}
		var err error
		sp.Packets, err = st.writeBufferAndSend(ctx, sp.Packets)
		if err != nil && err != io.EOF {
			log.Printf("writeBufferAndSend err:%v", err)
			return err
		}
		if len(sp.Packets) > 0 {
			if err := st.PushToLogChannel(ctx, sp); err != nil {
				return err
			}
			sp = nil
		}
		if err == io.EOF {
			return err
		}
	}
}

func (st *SendTask) sendState(ctx context.Context, state string) error {
	sp := st.newSendPacket()
	sp.State = state
	sp.Packets = sp.Packets[:0]
	return st.PushToLogChannel(ctx, sp)
}

func (st *SendTask) newSendPacket() *sendpacket.SendPacket {
	sp := st.GetSendPacket()
	sp.Datetime = time.Now().Unix()
	sp.User = st.User
	sp.Addr = st.Reader.RemoteAddr().String()
	sp.Db = st.DB
	sp.ConnectionID = st.ConnID
	sp.State = "est"
	return sp
}

func (st *SendTask) readFullMysqlPacket(ctx context.Context, buf []byte) (int, error) {
	size := 0
	for {
		if err := st.Reader.SetReadDeadline(time.Now().Add(st.Config.ConTimeout)); err != nil {
			return 0, err
		}
		nn, err := st.Reader.Read(buf)
		size += nn
		switch {
		case err == nil:
		case os.IsTimeout(err):
			select {
			case <-ctx.Done():
				return size, nil
			default:
			}
		case errors.Is(err, io.EOF):
			if size == 0 {
				return size, err
			}
			return size, io.ErrUnexpectedEOF
		case err != nil:
			log.Printf("read err:%s", err)
			return size, err
		}
		buf = buf[nn:]
		if len(buf) == 0 {
			return size, nil
		}
	}
}

func (st *SendTask) writeBufferAndSend(ctx context.Context, dst []byte) ([]byte, error) {
	dst = resizeSlice(dst, 4) //[]byte{0, 0, 0, 0}
	n, err := st.readFullMysqlPacket(ctx, dst)
	if err != nil {
		if n == 0 && err == io.EOF {
			return dst[:0], err
		}
		return dst[:0], fmt.Errorf("packet readFullMysqlPacket header err: %w n:%d", err, n)
	}

	length := int(uint32(dst[0]) | uint32(dst[1])<<8 | uint32(dst[2])<<16)
	//log.Printf("dst:%v length:%d len(dst):%d, cap(dst):%d,dst[:4]:%v", dst, length, len(dst), cap(dst), dst[:4])
	dst = resizeSlice(dst, length+4)
	//log.Printf("header:%v length:%d len(dst):%d, cap(dst):%d,dst[:4]:%v", header, length, len(dst), cap(dst), dst[:4])
	databuf := dst[4 : length+4]
	if n, err := st.readFullMysqlPacket(ctx, databuf); err != nil {
		return dst, fmt.Errorf("packet readFullMysqlPacket data err: %w n:%d want:%d", err, n, len(databuf))
	}
	if n, err := st.Writer.Write(dst[:length+4]); err != nil {
		return dst, fmt.Errorf("netWrite err: %w n:%d", err, n)
	}
	return dst[:length+4], nil
}

func resizeSlice(b []byte, size int) []byte {
	if cap(b) < size {
		return append(b, make([]byte, size-cap(b))...)
	}
	return b[:size]
}

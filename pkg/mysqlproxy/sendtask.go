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
	reader net.Conn
	writer net.Conn
	user   string
	db     string
	addr   string
	connID uint32
	Config *ProxyCfg
	LogWriter
}

func (st *SendTask) newSendPacket() *sendpacket.SendPacket {
	sp := st.GetSendPacket()
	sp.Datetime = time.Now().Unix()
	sp.User = st.user
	sp.Db = st.db
	sp.Addr = st.addr
	sp.ConnectionID = st.connID
	sp.State = "est"
	return sp
}

func (st *SendTask) Worker(ctx context.Context) {
	var sp *sendpacket.SendPacket
	defer func() {
		if sp != nil {
			st.PutSendPacket(sp)
		}
		st.CloseChannel()
	}()
	for {
		if sp == nil {
			sp = st.newSendPacket()
		}
		var err error
		sp.Packets, err = st.writeBufferAndSend(ctx, sp.Packets)
		if err != nil && err != io.EOF {
			log.Printf("writeBufferAndSend err:%v", err)
			return
		}
		if len(sp.Packets) > 0 {
			if err := st.PushToLogChannel(ctx, sp); err != nil {
				return
			}
		}
		if err == io.EOF {
			return
		}
	}
}

func (st *SendTask) readFullMysqlPacket(ctx context.Context, buf []byte) (int, error) {
	size := 0
	for {
		if err := st.reader.SetReadDeadline(time.Now().Add(st.Config.BufferFlushTime)); err != nil {
			return 0, err
		}
		nn, err := st.reader.Read(buf)
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

func resizeSlice(b []byte, size int) []byte {
	if cap(b) < size {
		return append(b, make([]byte, size-cap(b))...)
	}
	return b[:size]
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
	if n, err := st.writer.Write(dst[:length+4]); err != nil {
		return dst, fmt.Errorf("netWrite err: %w n:%d", err, n)
	}
	return dst[:length+4], nil
}

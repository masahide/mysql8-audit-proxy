package log

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
)

const (
	maxPacketSize = 0xffffff + 4
)

type auditLogWriter struct {
	filePath   string
	rotateTime time.Duration
	encode     func(w io.Writer, bbp *sendpacket.SendPacket) error

	dataPool    sync.Pool
	dataChannel chan *sendpacket.SendPacket
	file        *os.File
	gzipWriter  *gzip.Writer
	ticker      *time.Ticker
	latestFile  string
}

func NewAuditLogWriter(queue chan *sendpacket.SendPacket, filePath string, rotateTime time.Duration, t time.Time) (*auditLogWriter, error) {
	// Initialize auditLogWriter
	handler := &auditLogWriter{
		encode: sendpacket.EncodePacket,
		dataPool: sync.Pool{
			New: func() interface{} {
				sp := &sendpacket.SendPacket{}
				sp.Packets = make([]byte, maxPacketSize)
				return sp
			},
		},
		rotateTime:  rotateTime,
		filePath:    filePath,
		dataChannel: queue,
		ticker:      time.NewTicker(rotateTime),
	}
	// Create the initial file
	if err := handler.createFile(t); err != nil {
		return nil, err
	}

	return handler, nil
}

func (d *auditLogWriter) createFile(t time.Time) error {
	d.latestFile = time2Path(d.filePath, t)
	var err error
	//d.file, err = os.OpenFile(d.latestFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	d.file, err = os.OpenFile(d.latestFile, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		d.gzipWriter = gzip.NewWriter(d.file)
		d.gzipWriter.Write([]byte(fmtVersion)) // version
		return nil
	}
	if err != nil && os.IsExist(err) {
		d.file, err = os.OpenFile(d.latestFile, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		d.gzipWriter = gzip.NewWriter(d.file)
	}
	return err
}

func (d *auditLogWriter) writeDataToFile(data *sendpacket.SendPacket) error {
	if err := d.encode(d.gzipWriter, data); err != nil {
		return err
	}
	return nil
}

func (d *auditLogWriter) CloseChannel() {
	close(d.dataChannel)
}

func (d *auditLogWriter) closeFile() error {
	if d.gzipWriter != nil {
		if err := d.gzipWriter.Close(); err != nil {
			return err
		}
		d.gzipWriter = nil
	}
	if d.file != nil {
		err := d.file.Close()
		d.file = nil
		return err
	}
	return nil
}

func (d *auditLogWriter) receiveAndWrite(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case data, ok := <-d.dataChannel:
		//log.Printf("receive channel size:%d", len(d.dataChannel))
		if !ok {
			if err := d.closeFile(); err != nil {
				return err
			}
			return io.EOF
		}
		err := d.writeDataToFile(data)
		d.PutSendPacket(data)
		if err != nil {
			return err
		}

	case t := <-d.ticker.C:
		if err := d.closeFile(); err != nil {
			return err
		}

		if err := d.createFile(t); err != nil {
			return err
		}
	}
	return nil
}

func (d *auditLogWriter) LogWriteWorker(ctx context.Context) error {
	defer d.closeFile()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := d.receiveAndWrite(ctx)
		if err == nil {
			continue
		}
		if err == io.EOF {
			break
		}
	}
	return nil
}

func (d *auditLogWriter) GetLatestFilename() string { return d.latestFile }

func (d *auditLogWriter) PushToLogChannel(ctx context.Context, sp *sendpacket.SendPacket) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case d.dataChannel <- sp:
		//log.Printf("send channel size:%d", len(d.dataChannel))
	}
	return nil
}
func (d *auditLogWriter) PutSendPacket(b *sendpacket.SendPacket) {
	d.dataPool.Put(b)
}
func (d *auditLogWriter) GetSendPacket() *sendpacket.SendPacket {
	return d.dataPool.Get().(*sendpacket.SendPacket)
}

func Mkdir(filePath string) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir:\"%s\" error:%v", dir, err)
		}
	}
	return nil
}

/*
Usage example:
```
	filePath := "mysql-audit.%Y%m%d%H%M.log"
	log.Mkdir(filePath)
	q := make(chan *sendpacket.SendPacket, 1000)
	logHandler, err := log.NewAuditLogWriter(q, filePath, rotateTime, time.Now())
	wg:=sync.WaitGroup{}
	wg.Add(1)
	go func(){
		logHandler.LogWriteWorker(ctx)
		wg.Done()
	}()
	wg.Wait()
```
*/

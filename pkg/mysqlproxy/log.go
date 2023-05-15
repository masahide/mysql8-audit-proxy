package mysqlproxy

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type DataHandler struct {
	filePath   string
	rotateTime time.Duration

	dataPool    sync.Pool
	dataChannel chan *SendPacket
	file        *os.File
	gzipWriter  *gzip.Writer
	isFirst     bool
	ticker      *time.Ticker
}

func NewDataHandler(filePath string, rotateTime time.Duration, t time.Time) (*DataHandler, error) {
	// Initialize DataHandler
	handler := &DataHandler{
		dataPool: sync.Pool{
			New: func() interface{} {
				return &SendPacket{}
			},
		},
		rotateTime:  rotateTime,
		filePath:    filePath,
		dataChannel: make(chan *SendPacket, 10),
		ticker:      time.NewTicker(rotateTime),
	}
	// Create the initial file
	if err := handler.createFile(t); err != nil {
		return nil, err
	}

	return handler, nil
}

func (d *DataHandler) generateData(in *SendPacket) {
	data := d.dataPool.Get().(*SendPacket)
	*data = *in
	d.dataChannel <- data
}

func (d *DataHandler) createFile(t time.Time) error {
	filename := time2Path(d.filePath, t)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	gw := gzip.NewWriter(file)

	d.file = file
	d.gzipWriter = gw
	gw.Write([]byte{0, 0, 0, 1}) // version
	d.isFirst = true

	return nil
}

func (d *DataHandler) writeDataToFile(data *SendPacket) error {
	err := data.EncodeBebop(d.gzipWriter)
	if err != nil {
		return err
	}

	d.isFirst = false
	return nil
}

func (d *DataHandler) closeFile() error {
	if err := d.gzipWriter.Close(); err != nil {
		return err
	}
	return d.file.Close()
}

func (d *DataHandler) sendAndReceiveData(ctx context.Context) error {
	select {
	case <-ctx.Done():
		if err := d.closeFile(); err != nil {
			return err
		}
	case data := <-d.dataChannel:
		if err := d.writeDataToFile(data); err != nil {
			return err
		}
		d.dataPool.Put(data)

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

type FileReader struct {
	f    *os.File
	gr   *gzip.Reader
	errR *errorReader
}

func NewFileReader(filename string) (*FileReader, error) {
	fr := &FileReader{}
	var err error
	// Open the generated file
	fr.f, err = os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	// Create a gzip reader
	fr.gr, err = gzip.NewReader(fr.f)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}

	fr.errR = NewErrorReader(fr.gr)
	return fr, nil
}

func (fr *FileReader) ReadSendPacket() (*SendPacket, error) {
	v := &SendPacket{}
	err := v.DecodeBebop(fr.errR)
	if fr.errR.Error() != nil {
		return nil, fr.errR.Error()
	}
	return v, err
}
func (fr *FileReader) Close() {
	fr.gr.Close()
	fr.f.Close()
}

// /path/to/mysql-audit.%Y%m%d%H.log
func time2Path(p string, t time.Time) string {
	p = strings.Replace(p, "%Y", fmt.Sprintf("%04d", t.Year()), -1)
	p = strings.Replace(p, "%y", fmt.Sprintf("%02d", t.Year()%100), -1)
	p = strings.Replace(p, "%m", fmt.Sprintf("%02d", t.Month()), -1)
	p = strings.Replace(p, "%d", fmt.Sprintf("%02d", t.Day()), -1)
	p = strings.Replace(p, "%H", fmt.Sprintf("%02d", t.Hour()), -1)
	p = strings.Replace(p, "%M", fmt.Sprintf("%02d", t.Minute()), -1)
	p = strings.Replace(p, "%S", fmt.Sprintf("%02d", t.Second()), -1)
	return p
}

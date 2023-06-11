package log

import (
	"compress/gzip"
	"fmt"
	"os"

	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
)

type FileReader struct {
	f       *os.File
	gr      *gzip.Reader
	decoder *sendpacket.Decoder
	Decode  func(bbp *sendpacket.SendPacket) error
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
	version, err := checkFormat(fr.gr)
	//version, err := checkVersion(fr.f)
	if err != nil {
		return nil, err
	}
	switch version {
	case fmtVersion:
		break
	default:
		return nil, fmt.Errorf("version not match:%s", version)
	}

	fr.decoder = sendpacket.NewDecoder(fr.gr)
	fr.Decode = fr.decoder.DecodePacket
	return fr, nil
}

func (fr *FileReader) Close() {
	fr.gr.Close()
	fr.f.Close()
}

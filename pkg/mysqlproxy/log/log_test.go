package log

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
)

func TestSendAndReceiveData(t *testing.T) {
	testData := []sendpacket.SendPacket{
		{ConnectionID: 1, User: "Name1", Packets: []byte{0, 0}},
		{ConnectionID: 2, User: "Name2", Packets: []byte{0, 0}},
		{ConnectionID: 111, Cmd: strings.Repeat("4", 224340), Packets: bytes.Repeat([]byte{0}, 0)},
		{ConnectionID: 111, Cmd: strings.Repeat("4", 222002), Packets: bytes.Repeat([]byte{0}, 1)},
		{ConnectionID: 10000, Cmd: strings.Repeat("4", 2001), Packets: bytes.Repeat([]byte{0}, 10000)},
		{ConnectionID: 100000, Cmd: strings.Repeat("4", 2001), Packets: bytes.Repeat([]byte{0}, 100000)},
		{ConnectionID: 1000000, Cmd: strings.Repeat("4", 2001), Packets: bytes.Repeat([]byte{0}, 1000000)},
		//{ConnectionID: 1000001, Cmd: strings.Repeat("4", 2001), Packets: bytes.Repeat([]byte{0}, 0xffffff)},
		{ConnectionID: 3, User: strings.Repeat("Namex", 10000), Packets: []byte{0}},
	}

	tempDir, err := os.MkdirTemp("", "test_log_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	filePath := filepath.Join(tempDir, "test.%Y%m%d%H%M.log")
	// Initialize DataHandler
	q := make(chan *sendpacket.SendPacket, 1000)
	handler, err := NewAuditLogHandler(q, filePath, 1*time.Minute, time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, td := range testData {
		// Generate data and send to channel
		data := handler.GetSendPacket()
		*data = td
		handler.PushToLogChannel(context.Background(), data)
	}
	handler.CloseChannel()

	// receive data and write to file
	for {
		err := handler.receiveAndWrite(context.Background())
		if err == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		t.Fatalf("failed to send and receive SendPacket: %v", err)
	}

	fr, err := NewFileReader(handler.GetLatestFilename())
	if err != nil {
		t.Fatal(err)
	}
	// Close the file
	if err := handler.closeFile(); err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	t.Logf("filename:%s", handler.GetLatestFilename())
	defer fr.Close()
	for _, td := range testData {

		v := sendpacket.SendPacket{}
		err := fr.Decode(&v)
		if err != nil && err != io.EOF {
			t.Fatalf("failed to decode data: %v", err)
		}
		if diff := cmp.Diff(td, v); diff != "" {
			t.Errorf("conID:%d User value is mismatch (-tom +tom2):\n%s", td.ConnectionID, diff)
		}
	}
	v := sendpacket.SendPacket{}
	if err := fr.Decode(&v); err != io.EOF {
		t.Fatalf("errR.Error() is not EOF: %v", err)
	}
}

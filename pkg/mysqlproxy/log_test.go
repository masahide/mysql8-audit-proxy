package mysqlproxy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSendAndReceiveData(t *testing.T) {
	testData := []SendPacket{
		{ConnectionID: 1, User: "Name1", Packets: []byte{}},
		{ConnectionID: 2, User: "Name2", Packets: []byte{}},
		{ConnectionID: 2, User: "Name2", Packets: []byte{}},
		//{ConnectionID: 111, User: strings.Repeat("Name", 10000), Packets: bytes.Repeat([]byte{0}, 1*1024*1024)},
		{ConnectionID: 2, User: "Name2", Packets: []byte{}},
		{ConnectionID: 5, User: "Name5", Packets: []byte{}},
		//{ConnectionID: 5, User: strings.Repeat("Name5", 10000), Packets: []byte{}},
	}

	dir, err := os.MkdirTemp("", "test_log_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	filePath := filepath.Join(dir, "test.log")
	// Initialize DataHandler
	handler, err := NewDataHandler(filePath, 1*time.Minute, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, td := range testData {
		// Generate data and send to channel
		handler.generateData(&td)
		// Send and receive data
		if err := handler.sendAndReceiveData(context.Background()); err != nil {
			t.Fatalf("failed to send and receive SendPacket: %v", err)
		}
	}

	// Close the file
	if err := handler.closeFile(); err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	fr, err := NewFileReader(handler.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer fr.Close()
	/*
		for _, td := range testData {
			// Decode the data
			t.Logf("1.file:%s, Decode data: %v", fr.f.Name(), td)
			receivedData, err := fr.ReadSendPacket()
			if err != nil && err != io.EOF {
				t.Fatalf("failed to decode data: %v", err)
			}
			t.Logf("2.file:%s", fr.f.Name())
			// Check if the generated data is same as the received data
			if td.ConnectionID != receivedData.ConnectionID || td.User != receivedData.User {
				t.Errorf("received data does not match generated data")
			}
		}
		if _, err := fr.ReadSendPacket(); err != io.EOF {
			t.Fatalf("errR.Error() is not EOF: %v", err)
		}
	*/
}

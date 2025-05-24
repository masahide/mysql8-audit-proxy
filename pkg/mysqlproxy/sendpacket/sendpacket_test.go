package sendpacket

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEncodeDecode(t *testing.T) {

	testcase := []struct {
		name   string
		packet SendPacket
		want   string
	}{
		{
			name: "normal",
			packet: SendPacket{
				Datetime:     1234,
				ConnectionID: 1,
				User:         "XXXX",
				Db:           "XXXX",
				Addr:         "XXXX",
				State:        "XXXX",
				Err:          "xxxx",
				Cmd:          "abc",
				Packets:      []byte("abcabc"),
			},
		},
		{
			name: "empty",
			packet: SendPacket{
				Datetime:     0,
				ConnectionID: 1,
				User:         "",
				Db:           "",
				Addr:         "",
				State:        "",
				Err:          "",
				Cmd:          "",
				Packets:      []byte(""),
			},
		},
		{
			name: "big",
			packet: SendPacket{
				Datetime:     121111,
				ConnectionID: 1,
				User:         strings.Repeat("x", 2000),
				Db:           strings.Repeat("x", 2000),
				Addr:         strings.Repeat("x", 2000),
				State:        strings.Repeat("x", 2000),
				Err:          strings.Repeat("x", 2000),
				Cmd:          strings.Repeat("x", 20000),
				Packets:      bytes.Repeat([]byte("x"), 20000),
			},
		},
	}
	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			if err := EncodePacket(buf, &tc.packet); err != nil {
				t.Fatal(err)
			}
			encoded := buf.Bytes()
			read := bytes.NewBuffer(encoded)
			r := NewDecoder(read)
			res := SendPacket{Packets: make([]byte, maxBuf)}
			if err := r.DecodePacket(&res); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.packet, res); diff != "" {
				t.Errorf("User value is mismatch (-tom +tom2):\n%s", diff)
			}
		})
	}
	t.Run("loop", func(t *testing.T) {
		w := &bytes.Buffer{}
		for _, tc := range testcase {
			if err := EncodePacket(w, &tc.packet); err != nil {
				t.Fatal(err)
			}

		}
		encoded := w.Bytes()
		read := bytes.NewBuffer(encoded)
		r := NewDecoder(read)
		res := SendPacket{Packets: make([]byte, maxBuf)}
		for _, tc := range testcase {
			if err := r.DecodePacket(&res); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.packet, res); diff != "" {
				t.Errorf("User value is mismatch (-tom +tom2):\n%s", diff)
			}
		}
		t.Run("EOF", func(t *testing.T) {
			err := r.DecodePacket(&res)
			if err != io.EOF {
				t.Fatalf("err is not EOF but %v", err)
			}
		})
	})

}

func TestDecodePacketAliasing(t *testing.T) {
	originalPacket := SendPacket{
		Datetime:     1678886400,
		ConnectionID: 12345,
		User:         "test_user",
		Db:           "test_db",
		Addr:         "127.0.0.1:12345",
		State:        "test_state",
		Err:          "test_error_message",
		Cmd:          "test_command",
		Packets:      []byte{0x01, 0x02, 0x03, 0x04},
	}

	buf := &bytes.Buffer{}
	if err := EncodePacket(buf, &originalPacket); err != nil {
		t.Fatalf("EncodePacket failed: %v", err)
	}

	decoder := NewDecoder(buf)
	var decodedPacket SendPacket
	// Initialize Packets slice to a different capacity to check if DecodePacket reuses it or allocates a new one.
	// This is a more robust check against aliasing if the underlying slice is merely resized.
	decodedPacket.Packets = make([]byte, 0, len(originalPacket.Packets)+10)


	if err := decoder.DecodePacket(&decodedPacket); err != nil {
		t.Fatalf("DecodePacket failed: %v", err)
	}

	if !reflect.DeepEqual(originalPacket, decodedPacket) {
		t.Errorf("Decoded packet does not match original packet.\nOriginal: %+v\nDecoded:  %+v", originalPacket, decodedPacket)
	}

	// Further check to ensure Packets field was copied, not aliased.
	// Modify originalPacket.Packets and check if decodedPacket.Packets changes.
	// This is a direct test for aliasing.
	originalPacket.Packets[0] = 0xff
	if reflect.DeepEqual(originalPacket, decodedPacket) {
		t.Errorf("Decoded packet's Packets field is aliased to the original packet's Packets field.\nOriginal: %+v\nDecoded:  %+v", originalPacket, decodedPacket)
	}
	// Check that decodedPacket.Packets still holds the original data
	expectedPackets := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(decodedPacket.Packets, expectedPackets) {
		t.Errorf("decodedPacket.Packets was modified after originalPacket.Packets was modified. Expected: %v, Got: %v", expectedPackets, decodedPacket.Packets)
	}
}

package mysqlproxy

import (
	"bytes"
	"io"
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

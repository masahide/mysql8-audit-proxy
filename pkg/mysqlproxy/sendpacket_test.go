package mysqlproxy

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMarshalBebopTo(t *testing.T) {

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
			w := NewSpWriter(buf)
			if err := w.Encode(&tc.packet); err != nil {
				t.Fatal(err)
			}
			t.Logf("buf.Len()=%d", buf.Len())
			read := bytes.NewBuffer(buf.Bytes())
			r := NewSpReader(read)
			want := &SendPacket{Packets: make([]byte, maxBuf)}
			if err := r.Decode(want); err != nil {
				t.Fatal(err)
			}
			//bs := tc.packet.MarshalBebop()
			//t.Logf("len(bs)=%d", len(bs))
			//want, err := MakeSendPacketFromBytes(bs)
			if diff := cmp.Diff(tc.packet, want); diff != "" {
				t.Errorf("User value is mismatch (-tom +tom2):\n%s", diff)
			}
		})
	}

}

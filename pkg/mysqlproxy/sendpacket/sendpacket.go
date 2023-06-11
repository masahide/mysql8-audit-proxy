package sendpacket

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

const (
	maxBuf = 64 * 1024 * 1024
)

type SendPacket struct {
	Datetime     int64  `json:"time"` // unix time
	ConnectionID uint32 `json:"id,omitempty"`
	User         string `json:"user,omitempty"`
	Db           string `json:"db,omitempty"`
	Addr         string `json:"addr,omitempty"`
	State        string `json:"state,omitempty"`   // 5
	Err          string `json:"err,omitempty"`     // 6
	Packets      []byte `json:"packets,omitempty"` // 7
	Cmd          string `json:"cmd,omitempty"`
}

func writeBytes(w io.Writer, b []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func JsonEncodePacket(w io.Writer, bbp *SendPacket) error {
	return json.NewEncoder(w).Encode(bbp)
}

func EncodePacket(w io.Writer, bbp *SendPacket) error {
	if err := binary.Write(w, binary.LittleEndian, bbp.Datetime); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bbp.ConnectionID); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.User)); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.Db)); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.Addr)); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.State)); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.Err)); err != nil {
		return err
	}
	if err := writeBytes(w, []byte(bbp.Cmd)); err != nil {
		return err
	}
	if err := writeBytes(w, bbp.Packets); err != nil {
		return err
	}
	return nil
}

type Decoder struct {
	buf []byte
	r   io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		buf: make([]byte, 1024),
		r:   r,
	}
}

func (d *Decoder) readBytes(r io.Reader) ([]byte, error) {
	var len uint32
	if err := binary.Read(r, binary.LittleEndian, &len); err != nil {
		return nil, err
	}

	if uint32(cap(d.buf)) < len {
		d.buf = make([]byte, len)
	} else {
		d.buf = d.buf[:len]
	}

	if _, err := io.ReadFull(r, d.buf); err != nil {
		return nil, err
	}

	return d.buf, nil
}

func (d *Decoder) DecodePacket(bbp *SendPacket) error {
	if err := binary.Read(d.r, binary.LittleEndian, &bbp.Datetime); err != nil {
		return err
	}
	if err := binary.Read(d.r, binary.LittleEndian, &bbp.ConnectionID); err != nil {
		return err
	}

	var err error
	var data []byte
	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.User = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.Db = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.Addr = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.State = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.Err = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.Cmd = string(data)

	if data, err = d.readBytes(d.r); err != nil {
		return err
	}
	bbp.Packets = data

	return nil
}

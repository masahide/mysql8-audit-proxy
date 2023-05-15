package mysqlproxy

import (
	"encoding/binary"
	"io"

	"github.com/200sc/bebop"
	"github.com/200sc/bebop/iohelp"
)

const (
	maxBuf = 64 * 1024 * 1024
)

var _ bebop.Record = &SendPacket{}

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

func (bbp SendPacket) MarshalBebopTo(buf []byte) int {
	at := 0
	iohelp.WriteInt64Bytes(buf[at:], bbp.Datetime)
	at += 8
	iohelp.WriteUint32Bytes(buf[at:], bbp.ConnectionID)
	at += 4
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.User)))
	copy(buf[at+4:at+4+len(bbp.User)], []byte(bbp.User))
	at += 4 + len(bbp.User)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.Db)))
	copy(buf[at+4:at+4+len(bbp.Db)], []byte(bbp.Db))
	at += 4 + len(bbp.Db)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.Addr)))
	copy(buf[at+4:at+4+len(bbp.Addr)], []byte(bbp.Addr))
	at += 4 + len(bbp.Addr)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.State)))
	copy(buf[at+4:at+4+len(bbp.State)], []byte(bbp.State))
	at += 4 + len(bbp.State)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.Err)))
	copy(buf[at+4:at+4+len(bbp.Err)], []byte(bbp.Err))
	at += 4 + len(bbp.Err)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.Cmd)))
	copy(buf[at+4:at+4+len(bbp.Cmd)], []byte(bbp.Cmd))
	at += 4 + len(bbp.Cmd)
	iohelp.WriteUint32Bytes(buf[at:], uint32(len(bbp.Packets)))
	at += 4
	copy(buf[at:at+len(bbp.Packets)], bbp.Packets)
	at += len(bbp.Packets)
	return at
}

func (bbp *SendPacket) UnmarshalBebop(buf []byte) (err error) {
	at := 0
	if len(buf[at:]) < 8 {
		return io.ErrUnexpectedEOF
	}
	bbp.Datetime = iohelp.ReadInt64Bytes(buf[at:])
	at += 8
	if len(buf[at:]) < 4 {
		return io.ErrUnexpectedEOF
	}
	bbp.ConnectionID = iohelp.ReadUint32Bytes(buf[at:])
	at += 4
	bbp.User, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.User)
	bbp.Db, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.Db)
	bbp.Addr, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.Addr)
	bbp.State, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.State)
	bbp.Err, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.Err)
	bbp.Cmd, err = iohelp.ReadStringBytes(buf[at:])
	if err != nil {
		return err
	}
	at += 4 + len(bbp.Cmd)
	if len(buf[at:]) < 4 {
		return io.ErrUnexpectedEOF
	}
	bbp.Packets = make([]byte, iohelp.ReadUint32Bytes(buf[at:]))
	at += 4
	if len(buf[at:]) < len(bbp.Packets)*1 {
		return io.ErrUnexpectedEOF
	}
	copy(bbp.Packets, buf[at:at+len(bbp.Packets)])
	at += len(bbp.Packets)
	return nil
}

func (bbp SendPacket) EncodeBebop(iow io.Writer) (err error) {
	w := iohelp.NewErrorWriter(iow)
	iohelp.WriteInt64(w, bbp.Datetime)
	iohelp.WriteUint32(w, bbp.ConnectionID)
	iohelp.WriteUint32(w, uint32(len(bbp.User)))
	w.Write([]byte(bbp.User))
	iohelp.WriteUint32(w, uint32(len(bbp.Db)))
	w.Write([]byte(bbp.Db))
	iohelp.WriteUint32(w, uint32(len(bbp.Addr)))
	w.Write([]byte(bbp.Addr))
	iohelp.WriteUint32(w, uint32(len(bbp.State)))
	w.Write([]byte(bbp.State))
	iohelp.WriteUint32(w, uint32(len(bbp.Err)))
	w.Write([]byte(bbp.Err))
	iohelp.WriteUint32(w, uint32(len(bbp.Cmd)))
	w.Write([]byte(bbp.Cmd))
	iohelp.WriteUint32(w, uint32(len(bbp.Packets)))
	for _, elem := range bbp.Packets {
		iohelp.WriteByte(w, elem)
	}
	return w.Err
}

func (bbp *SendPacket) DecodeBebop(ior io.Reader) (err error) {
	r := iohelp.NewErrorReader(ior)
	bbp.Datetime = iohelp.ReadInt64(r)
	bbp.ConnectionID = iohelp.ReadUint32(r)
	bbp.User = iohelp.ReadString(r)
	bbp.Db = iohelp.ReadString(r)
	bbp.Addr = iohelp.ReadString(r)
	bbp.State = iohelp.ReadString(r)
	bbp.Err = iohelp.ReadString(r)
	bbp.Cmd = iohelp.ReadString(r)
	bbp.Packets = make([]byte, iohelp.ReadUint32(r))
	for i1 := range bbp.Packets {
		(bbp.Packets[i1]) = iohelp.ReadByte(r)
	}
	return r.Err
}

func (bbp SendPacket) Size() int {
	bodyLen := 0
	bodyLen += 8
	bodyLen += 4
	bodyLen += 4 + len(bbp.User)
	bodyLen += 4 + len(bbp.Db)
	bodyLen += 4 + len(bbp.Addr)
	bodyLen += 4 + len(bbp.State)
	bodyLen += 4 + len(bbp.Err)
	bodyLen += 4 + len(bbp.Cmd)
	bodyLen += 4
	bodyLen += len(bbp.Packets) * 1
	return bodyLen
}

func (bbp SendPacket) MarshalBebop() []byte {
	buf := make([]byte, bbp.Size())
	bbp.MarshalBebopTo(buf)
	return buf
}

func MakeSendPacket(r iohelp.ErrorReader) (SendPacket, error) {
	v := SendPacket{}
	err := v.DecodeBebop(r)
	return v, err
}

func MakeSendPacketFromBytes(buf []byte) (SendPacket, error) {
	v := SendPacket{}
	err := v.UnmarshalBebop(buf)
	return v, err
}

type spReader struct {
	buf []byte
	r   io.Reader
}

func NewSpReader(r io.Reader) *spReader {
	return &spReader{
		buf: make([]byte, maxBuf),
		r:   r,
	}
}

func (spr *spReader) ReadBytes(b []byte) (int, error) {
	if _, err := spr.r.Read(b[:4]); err != nil {
		return 0, err
	}
	s := binary.LittleEndian.Uint32(b[:4])
	return spr.r.Read(b[:s])
}
func (spr *spReader) ReadString(b []byte) (string, error) {
	n, err := spr.ReadBytes(b)
	if err != nil {
		return string(b), err
	}
	return string(b[:n]), nil
}
func (spr *spReader) Decode(bbp *SendPacket) error {
	if _, err := spr.r.Read(spr.buf[:8]); err != nil {
		return err
	}
	bbp.Datetime = int64(binary.LittleEndian.Uint64(spr.buf[:8]))

	if _, err := spr.r.Read(spr.buf[:4]); err != nil {
		return err
	}
	bbp.ConnectionID = binary.LittleEndian.Uint32(spr.buf[:4])
	var err error
	if bbp.User, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	if bbp.Db, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	if bbp.Addr, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	if bbp.State, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	if bbp.Err, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	if bbp.Cmd, err = spr.ReadString(spr.buf); err != nil {
		return err
	}
	_, err = spr.ReadBytes(bbp.Packets[:maxBuf])
	return err
}

type spWriter struct {
	buf []byte
	w   io.Writer
}

func NewSpWriter(w io.Writer) *spWriter {
	return &spWriter{
		buf: make([]byte, maxBuf),
		w:   w,
	}
}

func (spw *spWriter) writeBytes(b []byte) error {
	binary.LittleEndian.PutUint32(spw.buf[:4], uint32(len(b)))
	if _, err := spw.w.Write(spw.buf[:4]); err != nil {
		return err
	}
	_, err := spw.w.Write(b)
	return err
}

func (spw *spWriter) Encode(bbp *SendPacket) error {
	binary.LittleEndian.PutUint64(spw.buf[:8], uint64(bbp.Datetime))
	if _, err := spw.w.Write(spw.buf[:8]); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(spw.buf[:4], bbp.ConnectionID)
	if _, err := spw.w.Write(spw.buf[:4]); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.User)); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.Db)); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.Addr)); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.State)); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.Err)); err != nil {
		return err
	}
	if err := spw.writeBytes([]byte(bbp.Cmd)); err != nil {
		return err
	}
	if err := spw.writeBytes(bbp.Packets); err != nil {
		return err
	}
	return nil
}

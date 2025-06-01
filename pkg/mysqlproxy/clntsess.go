package mysqlproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/bits"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/masahide/mysql8-audit-proxy/pkg/timeoutnet"
)

type ClientSess struct {
	ClientMysql    *server.Conn
	TargetMysql    *client.Conn
	ProxySrv       *ProxySrv
	TargetNet      string
	TargetAddr     string
	TargetUser     string
	TargetPassword string
	TargetDB       string
}

func DumpResult(res *mysql.Result, err error) {
	if err != nil {
		log.Printf("execute query on target mysql err:%s", err)
		return
	}
	if res != nil && res.Resultset != nil {
		log.Printf("Result Status: %d, InsertId: %d, AffectedRows: %d", res.Status, res.InsertId, res.AffectedRows)
		for i, field := range res.Resultset.Fields {
			log.Printf("Field[%d]: %s", i, field.Name)
		}
		for rowIdx, row := range res.Resultset.Values {
			var vals []string
			for colIdx, val := range row {
				vals = append(vals, string(val.AsString()))
				log.Printf("Row[%d] Col[%d]: %s", rowIdx, colIdx, string(val.AsString()))
			}
		}
	}
}

func PrintCapability(capability uint32) {
	caps := make([]string, 0, bits.OnesCount32(capability))
	for capability != 0 {
		field := uint32(1 << bits.TrailingZeros32(capability))
		capability ^= field

		switch field {
		case mysql.CLIENT_LONG_PASSWORD:
			caps = append(caps, "CLIENT_LONG_PASSWORD")
		case mysql.CLIENT_FOUND_ROWS:
			caps = append(caps, "CLIENT_FOUND_ROWS")
		case mysql.CLIENT_LONG_FLAG:
			caps = append(caps, "CLIENT_LONG_FLAG")
		case mysql.CLIENT_CONNECT_WITH_DB:
			caps = append(caps, "CLIENT_CONNECT_WITH_DB")
		case mysql.CLIENT_NO_SCHEMA:
			caps = append(caps, "CLIENT_NO_SCHEMA")
		case mysql.CLIENT_COMPRESS:
			caps = append(caps, "CLIENT_COMPRESS")
		case mysql.CLIENT_ODBC:
			caps = append(caps, "CLIENT_ODBC")
		case mysql.CLIENT_LOCAL_FILES:
			caps = append(caps, "CLIENT_LOCAL_FILES")
		case mysql.CLIENT_IGNORE_SPACE:
			caps = append(caps, "CLIENT_IGNORE_SPACE")
		case mysql.CLIENT_PROTOCOL_41:
			caps = append(caps, "CLIENT_PROTOCOL_41")
		case mysql.CLIENT_INTERACTIVE:
			caps = append(caps, "CLIENT_INTERACTIVE")
		case mysql.CLIENT_SSL:
			caps = append(caps, "CLIENT_SSL")
		case mysql.CLIENT_IGNORE_SIGPIPE:
			caps = append(caps, "CLIENT_IGNORE_SIGPIPE")
		case mysql.CLIENT_TRANSACTIONS:
			caps = append(caps, "CLIENT_TRANSACTIONS")
		case mysql.CLIENT_RESERVED:
			caps = append(caps, "CLIENT_RESERVED")
		case mysql.CLIENT_SECURE_CONNECTION:
			caps = append(caps, "CLIENT_SECURE_CONNECTION")
		case mysql.CLIENT_MULTI_STATEMENTS:
			caps = append(caps, "CLIENT_MULTI_STATEMENTS")
		case mysql.CLIENT_MULTI_RESULTS:
			caps = append(caps, "CLIENT_MULTI_RESULTS")
		case mysql.CLIENT_PS_MULTI_RESULTS:
			caps = append(caps, "CLIENT_PS_MULTI_RESULTS")
		case mysql.CLIENT_PLUGIN_AUTH:
			caps = append(caps, "CLIENT_PLUGIN_AUTH")
		case mysql.CLIENT_CONNECT_ATTRS:
			caps = append(caps, "CLIENT_CONNECT_ATTRS")
		case mysql.CLIENT_PLUGIN_AUTH_LENENC_CLIENT_DATA:
			caps = append(caps, "CLIENT_PLUGIN_AUTH_LENENC_CLIENT_DATA")
		case mysql.CLIENT_CAN_HANDLE_EXPIRED_PASSWORDS:
			caps = append(caps, "CLIENT_CAN_HANDLE_EXPIRED_PASSWORDS")
		case mysql.CLIENT_SESSION_TRACK:
			caps = append(caps, "CLIENT_SESSION_TRACK")
		case mysql.CLIENT_DEPRECATE_EOF:
			caps = append(caps, "CLIENT_DEPRECATE_EOF")
		case mysql.CLIENT_OPTIONAL_RESULTSET_METADATA:
			caps = append(caps, "CLIENT_OPTIONAL_RESULTSET_METADATA")
		case mysql.CLIENT_ZSTD_COMPRESSION_ALGORITHM:
			caps = append(caps, "CLIENT_ZSTD_COMPRESSION_ALGORITHM")
		case mysql.CLIENT_QUERY_ATTRIBUTES:
			caps = append(caps, "CLIENT_QUERY_ATTRIBUTES")
		case mysql.MULTI_FACTOR_AUTHENTICATION:
			caps = append(caps, "MULTI_FACTOR_AUTHENTICATION")
		case mysql.CLIENT_CAPABILITY_EXTENSION:
			caps = append(caps, "CLIENT_CAPABILITY_EXTENSION")
		case mysql.CLIENT_SSL_VERIFY_SERVER_CERT:
			caps = append(caps, "CLIENT_SSL_VERIFY_SERVER_CERT")
		case mysql.CLIENT_REMEMBER_OPTIONS:
			caps = append(caps, "CLIENT_REMEMBER_OPTIONS")
		default:
			caps = append(caps, fmt.Sprintf("(%d)", field))
		}
	}
	log.Printf("capabilitystring: %s", strings.Join(caps, "|"))
}

// ClntSess methods
// ConnectToMySQL connect to mysql target server
func (c *ClientSess) ConnectToMySQL(ctx context.Context) error {
	// PrintCapability(c.ClientMysql.Capability())
	dialer := &net.Dialer{}
	clientDialer := dialer.DialContext
	TargetConn, err := client.ConnectWithDialer(ctx,
		c.TargetNet, c.TargetAddr, c.TargetUser, c.TargetPassword, c.TargetDB, clientDialer,
		func(con *client.Conn) error {
			cap := c.ClientMysql.Capability() | mysql.CLIENT_LOCAL_FILES
			// PrintCapability(cap)
			con.SetCapability(cap)
			con.UnsetCapability(mysql.CLIENT_QUERY_ATTRIBUTES) // disable CLIENT_QUERY_ATTRIBUTES for now
			return nil
		},
	)
	if err != nil {
		log.Printf("connect to mysql target err:%s", err)
		return err
	}
	// TargetConn.UnsetCapability(mysql.CLIENT_QUERY_ATTRIBUTES) // disable CLIENT_QUERY_ATTRIBUTES for now
	// ci := TargetConn.Conn.Conn
	// TargetConn.Conn.Conn = WrapConn(ci, "targetConn:") // debug wrapper
	// DumpResult(TargetConn.Execute("select @@version, @@version_comment, @@version_compile_os, @@version_compile_machine"))

	c.TargetMysql = TargetConn
	return nil
}

func (c *ClientSess) Proxy(ctx context.Context) {
	cctx, cancel := context.WithCancel(ctx)
	clientWriter := &timeoutnet.TimeoutWriter{
		Conn:    c.ClientMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
	}
	targetReader := &timeoutnet.TimeoutReader{
		Conn:    c.TargetMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
	}
	clientReader := &timeoutnet.TimeoutReader{
		Conn:    c.ClientMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
	}
	targetWriter := &timeoutnet.TimeoutWriter{
		Conn:    c.TargetMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
	}
	st := &SendTask{
		Reader:    clientReader,
		Writer:    targetWriter,
		User:      c.TargetUser,
		DB:        c.TargetDB,
		Addr:      c.TargetAddr,
		ConnID:    c.ClientMysql.ConnectionID(),
		Config:    c.ProxySrv.Config,
		LogWriter: c.ProxySrv.AuditLogWriter,
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		//_, err := CopyDebug("clientWriter:", clientWriter, targetReader)
		_, err := io.Copy(clientWriter, targetReader)
		if err != nil {
			log.Printf("clientWriter err:%v", err)
		}
		cancel()
		wg.Done()
	}()
	// TODO:
	// log.Printf("start worker for user:%s, db:%s, addr:%s, connID:%d", c.TargetUser, c.TargetDB, c.TargetAddr, c.ClientMysql.ConnectionID())
	err := st.Worker(ctx)
	//_, err := CopyDebug("targetWriter:", targetWriter, clientReader)
	if err != nil && err != context.Canceled && err != io.EOF {
		log.Printf("targetWriter err:%v", err)
	}
	cancel()
	wg.Wait()
	if err := c.TargetMysql.Close(); err != nil {
		log.Printf("targetMysql close err:%v", err)
	}
}

type DebugWriter struct {
	W      io.Writer // 実際に書き込む先
	Prefix string    // 行頭に付けるプレフィックス（例: "clientWrite:"）
	offset int64     // 累積オフセット（連続 Write でも行番号がつながるように）
}

// Write implements io.Writer.
func (d *DebugWriter) Write(p []byte) (int, error) {
	d.dump(p)
	n, err := d.W.Write(p)
	d.offset += int64(n)
	return n, err
}

// CopyDebug は io.Copy 相当のヘルパー。
// dst 側に DebugWriter をかませてプレフィックス付きダンプを行う。
func CopyDebug(prefix string, dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(&DebugWriter{W: dst, Prefix: prefix}, src)
}

// ---- 内部実装 -------------------------------------------------------------

func (d *DebugWriter) dump(b []byte) {
	const chunk = 16
	for offset := 0; offset < len(b); offset += chunk {
		end := offset + chunk
		if end > len(b) {
			end = len(b)
		}
		line := b[offset:end]

		// hex 部
		var hexParts []string
		for i := 0; i < chunk; i++ {
			if i < len(line) {
				hexParts = append(hexParts, fmt.Sprintf("%02x", line[i]))
			} else {
				hexParts = append(hexParts, "  ")
			}
			if i == 7 { // 8 byte で空白を挟む
				hexParts = append(hexParts, "")
			}
		}

		// ASCII 部
		var asciiParts []byte
		for _, c := range line {
			if c >= 0x20 && c <= 0x7e {
				asciiParts = append(asciiParts, c)
			} else {
				asciiParts = append(asciiParts, '.')
			}
		}

		fmt.Fprintf(os.Stderr, "%s %08x  %-48s |%s|\n",
			d.Prefix,
			d.offset+int64(offset),
			strings.Join(hexParts, " "),
			string(asciiParts),
		)
	}
}

type Conn interface {
	net.Conn
	SetReadDeadline(t time.Time) error  // go-mysql 側で呼ばれることが多いので proxy
	SetWriteDeadline(t time.Time) error // 同上
}

// DebugConn は書き込みデバッグ用の薄いラッパ。
type DebugConn struct {
	Conn          Conn
	Prefix        string
	offset        int64 // 連続 Write でオフセット継続
	dumpDest      io.Writer
	maxDumpPerBuf int // 0 なら制限なし
}

// WrapConn で包むだけで OK。
func WrapConn(c Conn, prefix string) *DebugConn {
	return &DebugConn{
		Conn:     c,
		Prefix:   prefix,
		dumpDest: os.Stderr,
	}
}

// --- net.Conn インタフェースの実装 ---

func (d *DebugConn) Read(b []byte) (int, error) {
	return d.Conn.Read(b) // 受信側もダンプしたいならここで dump を呼ぶ
}

func (d *DebugConn) Write(b []byte) (int, error) {
	// ① 先にダンプ
	d.dump(b)

	// ② 実際に送信
	n, err := d.Conn.Write(b)
	d.offset += int64(n)
	return n, err
}

// 以下は net.Conn の他メソッドを素通し
func (d *DebugConn) Close() error                       { return d.Conn.Close() }
func (d *DebugConn) LocalAddr() net.Addr                { return d.Conn.LocalAddr() }
func (d *DebugConn) RemoteAddr() net.Addr               { return d.Conn.RemoteAddr() }
func (d *DebugConn) SetDeadline(t time.Time) error      { return d.Conn.SetDeadline(t) }
func (d *DebugConn) SetReadDeadline(t time.Time) error  { return d.Conn.SetReadDeadline(t) }
func (d *DebugConn) SetWriteDeadline(t time.Time) error { return d.Conn.SetWriteDeadline(t) }

// --- 内部：hex+ASCII ダンプ ---

func (d *DebugConn) dump(buf []byte) {
	const chunk = 16

	if d.maxDumpPerBuf > 0 && len(buf) > d.maxDumpPerBuf {
		buf = buf[:d.maxDumpPerBuf]
	}

	for off := 0; off < len(buf); off += chunk {
		end := off + chunk
		if end > len(buf) {
			end = len(buf)
		}
		line := buf[off:end]

		// hex 部
		var hexParts []string
		for i := 0; i < chunk; i++ {
			if i < len(line) {
				hexParts = append(hexParts, fmt.Sprintf("%02x", line[i]))
			} else {
				hexParts = append(hexParts, "  ")
			}
			if i == 7 { // 8byte ごとに空白
				hexParts = append(hexParts, "")
			}
		}

		// ASCII 部
		var ascii []byte
		for _, c := range line {
			if c >= 0x20 && c <= 0x7e {
				ascii = append(ascii, c)
			} else {
				ascii = append(ascii, '.')
			}
		}

		fmt.Fprintf(d.dumpDest, "%s %08x  %-48s |%s|\n",
			d.Prefix,
			d.offset+int64(off),
			strings.Join(hexParts, " "),
			string(ascii),
		)
	}
}

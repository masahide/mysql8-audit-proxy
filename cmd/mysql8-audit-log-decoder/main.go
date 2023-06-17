package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	proxylog "github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/log"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	showVer = flag.Bool("version", false, "Show version")
)

func main() {
	flag.Parse()
	if *showVer {
		// nolint: errcheck
		fmt.Printf("version: %v\ncommit: %v\nbuilt_at: %v\n", version, commit, date)
		return
	}
	for _, arg := range flag.Args() {
		err := filePrint(arg)
		if err != nil {
			log.Printf("cannot print file:%s, err:%s", arg, err)
		}
	}
}

func filePrint(filename string) error {
	r, err := proxylog.NewFileReader(filename)
	if err != nil {
		return err
	}
	defer r.Close()
	bp := sendpacket.SendPacket{}
	for {
		err := r.Decode(&bp)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		os.Stdout.Write(fmtJSON(formatPacket(bp)))
		/*
			b, err := trim(bp.Packets)
			if err != nil {
				log.Printf("cannot trim err:%s packet:%s", err, fmtJSON(bp))
			} else {
				fmt.Printf("%s: size:%d, %v\n", time.Unix(bp.Datetime, 0).Format("2006-01-02 15:04:05"), len(b), b)
			}
		*/
	}
	return nil
}

func fmtJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("cannot print json:%s", err)
		return []byte{}
	}
	return append([]byte(b), byte('\n'))
}

type packet struct {
	Datetime     time.Time `json:"time"`
	ConnectionID uint32    `json:"con_id,omitempty"`
	User         string    `json:"user,omitempty"`
	Db           string    `json:"db,omitempty"`
	Addr         string    `json:"addr,omitempty"`
	State        string    `json:"state,omitempty"`
	Err          string    `json:"err,omitempty"`
	Packets      []byte    `json:"packets,omitempty"`
	Cmd          string    `json:"cmd,omitempty"`
}

func formatPacket(sp sendpacket.SendPacket) (res packet) {
	res = packet{
		Datetime:     time.Unix(sp.Datetime, 0),
		ConnectionID: sp.ConnectionID,
		User:         sp.User,
		Db:           sp.Db,
		Addr:         sp.Addr,
		State:        sp.State,
		Err:          sp.Err,
	}
	data, err := trim(sp.Packets)
	if err != nil {
		res.Packets = sp.Packets
		return
	}
	if len(data) == 0 {
		res.Packets = sp.Packets
		return
	}
	cmd := data[0]
	data = data[1:]
	switch cmd {
	case mysql.COM_QUIT:
		res.Cmd = "quit"
		res.Packets = nil
	case mysql.COM_QUERY:
		res.Cmd = string(data)
		res.Packets = nil
	case mysql.COM_PING:
		res.Cmd = "ping"
		res.Packets = nil
	case mysql.COM_INIT_DB:
		res.Cmd = "use " + string(data)
		res.Packets = nil
	case mysql.COM_FIELD_LIST:
		index := bytes.IndexByte(data, 0x00)
		table := string(data[0:index])
		res.Cmd = "fieldList " + table
		wildcard := string(data[index+1:])
		if len(wildcard) > 0 {
			res.Cmd = res.Cmd + " " + wildcard
		}
		res.Packets = nil
	case mysql.COM_STMT_PREPARE:
		res.Cmd = "stmt_prepare"
		res.Packets = sp.Packets
	case mysql.COM_STMT_EXECUTE:
		res.Cmd = "stmt_execute"
		res.Packets = sp.Packets
	case mysql.COM_STMT_CLOSE:
		res.Cmd = "stmt_close"
		res.Packets = sp.Packets
	case mysql.COM_STMT_SEND_LONG_DATA:
		res.Cmd = "stmt_send_long_data"
		res.Packets = sp.Packets
	case mysql.COM_STMT_RESET:
		res.Cmd = "stmt_reset"
		res.Packets = sp.Packets
	case mysql.COM_SET_OPTION:
		res.Cmd = "set_option"
		res.Packets = sp.Packets
	case mysql.COM_REGISTER_SLAVE:
		res.Cmd = "register_slave"
		res.Packets = sp.Packets
	case mysql.COM_BINLOG_DUMP:
		res.Cmd = "binlog_dump"
		res.Packets = sp.Packets
	case mysql.COM_BINLOG_DUMP_GTID:
		res.Cmd = "binlog_dump"
		res.Packets = sp.Packets
	default:
		res.Cmd = GetComName(cmd)
		res.Packets = sp.Packets
	}
	return
}

func trim(dst []byte) ([]byte, error) {
	if len(dst) < 4 {
		return nil, fmt.Errorf("dst too small %v", dst)
	}
	return dst[4:], nil
}

package mysqlproxy

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

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

// ClntSess methods
// ConnectToMySQL connect to mysql target server
func (c *ClientSess) ConnectToMySQL(ctx context.Context) error {
	dialer := &net.Dialer{}
	clientDialer := dialer.DialContext
	capLocalFiles := c.ClientMysql.Capability() & mysql.CLIENT_LOCAL_FILES
	TargetConn, err := client.ConnectWithDialer(ctx,
		c.TargetNet, c.TargetAddr, c.TargetUser, c.TargetPassword, c.TargetDB, clientDialer,
		func(con *client.Conn) {
			con.SetCapability(capLocalFiles)
		},
	)
	if err != nil {
		log.Printf("connect to mysql target err:%s", err)
		return err
	}
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
		_, err := io.Copy(clientWriter, targetReader)
		if err != nil {
			log.Printf("clientWriter err:%v", err)
		}
		cancel()
		wg.Done()
	}()
	// TODO:
	err := st.Worker(ctx)
	//_, err := io.Copy(targetWriter, clientReader)
	if err != nil && err != context.Canceled && err != io.EOF {
		log.Printf("targetWriter err:%v", err)
	}
	cancel()
	wg.Wait()
	if err := c.TargetMysql.Close(); err != nil {
		log.Printf("targetMysql close err:%v", err)
	}
}

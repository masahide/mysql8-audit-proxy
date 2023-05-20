package mysqlproxy

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/masahide/mysql8-audit-proxy/pkg/timeoutnet"
)

type ClientSess struct {
	ClientMysql *server.Conn
	TargetMysql *client.Conn
	ProxySrv    *ProxySrv
}

// ClntSess methods
// ConnectToMySQL connect to mysql target server
func (c *ClientSess) ConnectToMySQL() error {
	dialer := &net.Dialer{}
	clientDialer := dialer.DialContext
	ctx := context.Background()
	TargetConn, err := client.ConnectWithDialer(
		ctx,
		c.ProxySrv.Config.TargetNet,
		net.JoinHostPort(c.ProxySrv.Config.TargetAddr, c.ProxySrv.Config.TargetPort),
		c.ProxySrv.Config.TargetUser,
		c.ProxySrv.Config.TargetPass,
		c.ProxySrv.Config.TargetDB,
		clientDialer)
	if err != nil {
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
	}
	targetWriter := &timeoutnet.TimeoutWriter{
		Conn:    c.TargetMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
	}
	clientReader := &timeoutnet.TimeoutReader{
		Conn:    c.ClientMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
	}
	targetReader := &timeoutnet.TimeoutReader{
		Conn:    c.TargetMysql.Conn,
		Timeout: c.ProxySrv.Config.ConTimeout,
		Ctx:     cctx,
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
	_, err := io.Copy(targetWriter, clientReader)
	if err != nil {
		log.Printf("targetWrit err:%v", err)
	}
	cancel()
	wg.Wait()
	if err := c.TargetMysql.Close(); err != nil {
		log.Printf("targetMysql close err:%v", err)
	}
}

func (c *ClientSess) testSendDataToSrv() {
	if err := c.TargetMysql.Ping(); err != nil {
		log.Fatal(err)
	}

	// Select
	r, err := c.TargetMysql.Execute(`select * from user limit 1`)
	// Close result for reuse memory (it's not necessary but very useful)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	// Direct access to fields
	log.Printf("status: %d", r.Status)
	log.Printf("field count: %v", r.FieldNames)
	/*
		for _, row := range r.Values {
			for _, val := range row {
				v := val.Value() // interface{}
				log.Printf("value: %v", v)
			}
		}
	*/

	db := c.TargetMysql.GetDB()
	log.Printf("client DB:%s", db)
	for {
		err = c.ClientMysql.HandleCommand()
		if err != nil {
			log.Printf(`Could not handle command: %v`, err)
			break
		}
	}

}
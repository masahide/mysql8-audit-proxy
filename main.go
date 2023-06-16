package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy"
	proxylog "github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/log"
	"github.com/masahide/mysql8-audit-proxy/pkg/mysqlproxy/sendpacket"
	"github.com/masahide/mysql8-audit-proxy/pkg/serverconfig"
)

type Specification struct {
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:":3306"`
	AdminUser     string `default:"admin"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	svConfMng := serverconfig.NewManager(confDir)
	proxyConf := &mysqlproxy.ProxyCfg{}
	if err := envconfig.Process("", proxyConf); err != nil {
		log.Fatal(err)
	}
	if proxyConf.Debug {
		log.Printf("serverConfig %s", svConfMng.PrintPathInfo())
		log.Printf("proxyConfig:\n%s\n", dumpJSON(proxyConf))
	}
	q := make(chan *sendpacket.SendPacket, 1000)
	logHandler, err := proxylog.NewAuditLogWriter(q, proxyConf.LogFileName, proxyConf.RotateTime, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	p := &mysqlproxy.ProxySrv{
		AuditLogWriter: logHandler,
		SvConfMng:      svConfMng,
		Config:         proxyConf,
	}

	pctx, cancel := context.WithCancel(context.Background())
	ctx, stop := signal.NotifyContext(pctx, os.Interrupt)
	defer stop()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := logHandler.LogWriteWorker(ctx)
		log.Println("logWriteWorker stopped")
		if err != nil && err != context.Canceled {
			log.Printf("Proxy error: %v", err)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		err := p.Start(ctx)
		log.Println("proxy stopped")
		if err != nil && err != context.Canceled {
			log.Printf("Proxy error: %v", err)
		}
		logHandler.CloseChannel()
		cancel()
		wg.Done()
	}()
	wg.Wait()
}

func dumpJSON(v any) string {
	s, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(s)
}

/*
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var s Specification
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	l, err := net.Listen("tcp", s.ListenAddress)
	if err != nil {
		log.Fatal(err)
	}
	pemConf := generatepem.Config{}
	os.Setenv("HOST", "localhost")
	if err := envconfig.Process("", &pemConf); err != nil {
		log.Fatal(err)
	}
	caPems, serverPems, err := generatepem.Generate(pemConf)
	if err != nil {
		log.Fatal(err)
	}
	// user either the in-memory credential provider or the remote credential provider (you can implement your own)

	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	mng := serverconfig.NewManager(confDir)
	remoteProvider := mysqlproxy.NewConfigProvider(mng)
	var tlsConf = server.NewServerTLSConfig(
		[]byte(caPems.Cert),
		[]byte(serverPems.Cert),
		[]byte(serverPems.Key),
		tls.VerifyClientCertIfGiven,
	)
	clt := &client{
		serverPems:     serverPems,
		tlsConf:        tlsConf,
		remoteProvider: remoteProvider,
		mng:            mng,
		s:              s,
	}
	for {
		netConn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go clt.run(netConn)
	}
}

type client struct {
	serverPems     generatepem.Pems
	tlsConf        *tls.Config
	remoteProvider *mysqlproxy.ConfigProvider
	mng            *serverconfig.Manager
	s              Specification
}

func (c *client) run(netConn net.Conn) {
	defer netConn.Close()
	// Create a connection with user root and an empty password.
	// You can use your own handler to handle command here.
	svr := server.NewServer(
		"8.0.12",
		mysql.DEFAULT_COLLATION_ID,
		mysql.AUTH_CACHING_SHA2_PASSWORD,
		[]byte(c.serverPems.Public), c.tlsConf,
	)
	//conn, err := server.NewConn(c, "root", "fugga", server.EmptyHandler{})
	chandler := serverconfig.NewConfigHandler(c.mng)
	conn, err := server.NewCustomizedConn(netConn, svr, c.remoteProvider, chandler)
	if err != nil {
		log.Printf("Connection error: %v", err)
		return
	}
	defer conn.Close()
	user := conn.GetUser()
	log.Printf("user: %s", user)

	if user == c.s.AdminUser {
		for {
			err = conn.HandleCommand()
			if err != nil {
				log.Printf(`Could not handle command: %v`, err)
				break
			}
		}
		return
	}
	pw, err := c.remoteProvider.GetPassword(user)
	if err != nil {
		log.Printf("user:\"%s\" GetPassword error: %v", user, err)
		return
	}
	log.Printf("user:\"%s\" pw:\"%s\"", user, pw)
}
*/

/*
	user: 'user@host:port'
	pass: 'xxxx'

	user :'user@host'
	user :'user@abc.*'
	user :'user@.*'
*/

/*
	dialer := &net.Dialer{}
	clientDialer := dialer.DialContext
	ctx := context.Background()
	clientConn, err := client.ConnectWithDialer(ctx, "tcp", "localhost:3306", user, "PwTest01", "mysql", clientDialer)

	if err := clientConn.Ping(); err != nil {
		log.Fatal(err)
	}
	// Select
	r, err := clientConn.Execute(`select * from user limit 1`)
	// Close result for reuse memory (it's not necessary but very useful)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	// Direct access to fields
	log.Printf("status: %d", r.Status)
	log.Printf("field count: %v", r.FieldNames)

	//	for _, row := range r.Values {
	//		for _, val := range row {
	//			v := val.Value() // interface{}
	//			log.Printf("value: %v", v)
	//		}
	//	}

	db := clientConn.GetDB()
	log.Printf("client DB:%s", db)
*/

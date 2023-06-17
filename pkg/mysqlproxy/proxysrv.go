package mysqlproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/generatepem"
	"github.com/masahide/mysql8-audit-proxy/pkg/serverconfig"
)

// ProxySrv is  the main struct for the proxy server
type ProxySrv struct {
	listenSock net.Listener
	tlsConf    *tls.Config
	serverPems generatepem.Pems

	AuditLogWriter LogWriter
	SvConfMng      *serverconfig.Manager
	Config         *ProxyCfg
}

func (p *ProxySrv) Start(ctx context.Context) error {
	if err := p.createListener(); err != nil {
		return err
	}
	p.acceptClntConn(ctx)
	return nil
}

// ProxySrv methods
// createListener creates a listening socket
func (p *ProxySrv) createListener() error {
	listener, err := net.Listen(p.Config.ProxyListentNet, p.Config.ProxyListenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listening socket: %v", err)
	}
	p.listenSock = listener
	log.Printf("Proxy server listening on %s\n", p.Config.ProxyListenAddr)

	pemConf := generatepem.Config{}
	os.Setenv("HOST", "localhost")
	if err := envconfig.Process("", &pemConf); err != nil {
		log.Fatal(err)
	}
	caPems, serverPems, err := generatepem.Generate(pemConf)
	if err != nil {
		log.Fatal(err)
	}
	p.serverPems = serverPems
	p.tlsConf = server.NewServerTLSConfig(
		[]byte(caPems.Cert),
		[]byte(serverPems.Cert),
		[]byte(serverPems.Key),
		tls.VerifyClientCertIfGiven,
	)
	return nil
}
func (p *ProxySrv) acceptClntConn(ctx context.Context) {
	go func() {
		<-ctx.Done()
		p.listenSock.Close()
	}()
	for {
		conn, err := p.listenSock.Accept()
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err != nil {
			log.Printf("error accepting client connection: %v\n", err)
			continue
		}
		log.Printf("Accepted connection from %s", conn.RemoteAddr().String())
		// Handle client connection (e.g., create a new ClntSess and process the connection)
		go p.sessionWorker(ctx, conn)
	}
}

func (p *ProxySrv) sessionWorker(ctx context.Context, netConn net.Conn) {
	chandler := serverconfig.NewConfigHandler(p.SvConfMng)
	defer netConn.Close()
	svr := server.NewServer(
		"8.0.12_mysql-audit-proxy",
		mysql.DEFAULT_COLLATION_ID,
		mysql.AUTH_CACHING_SHA2_PASSWORD,
		[]byte(p.serverPems.Public), p.tlsConf,
	)
	remoteProvider := NewConfigProvider(p.SvConfMng)
	mysqlConn, err := server.NewCustomizedConn(netConn, svr, remoteProvider, chandler)
	if err != nil {
		log.Printf("Connection error: %v", err)
		return
	}
	defer func() {
		if !mysqlConn.Closed() {
			mysqlConn.Close()
		}
	}()

	user := mysqlConn.GetUser()
	log.Printf("user: %s", user)
	if user == p.Config.AdminUser {
		p.admin(mysqlConn)
		return
	}
	targetUser, targetAddr, targetPasswrd := getTargetInfo(user)
	if len(targetPasswrd) == 0 {
		targetPasswrd, _, _ = remoteProvider.GetCredential(user)
	}
	targetAddr = addPort(targetAddr)
	sess := &ClientSess{
		ClientMysql:    mysqlConn,
		TargetNet:      "tcp",
		TargetAddr:     targetAddr,
		TargetUser:     targetUser,
		TargetPassword: targetPasswrd,
		TargetDB:       chandler.GetDB(),
		ProxySrv:       p,
	}
	err = sess.ConnectToMySQL(ctx)
	if err != nil {
		log.Printf("error: connect to mysql target:%s err: %v", targetAddr, err)
		return
	}
	sess.Proxy(ctx)

}

// addPort - Add the "3306" port if the hostname does not indicate a port number. However, if the host name is empty, it will be localhost
func addPort(s string) string {
	if len(s) == 0 {
		s = "localhost:3306"
	}
	ss := strings.Split(s, ":")
	if len(ss) == 1 {
		return s + ":3306"
	}
	return s
}

func (p *ProxySrv) admin(mysqlConn *server.Conn) {
	for {
		if err := mysqlConn.HandleCommand(); err != nil {
			log.Printf(`Could not handle command: %v`, err)
			return
		}
	}
}

func getUserPass(s string) (user string, pass string) {
	substrings := strings.Split(s, ":")
	if len(substrings) != 2 {
		return s, ""
	}
	return substrings[0], substrings[1]
}
func getTargetInfo(s string) (user string, server string, pass string) {
	substrings := strings.Split(s, "@")
	if len(substrings) != 2 {
		user, pass = getUserPass(s)
		return
	}
	user, pass = getUserPass(substrings[0])
	server = substrings[1]
	return
}

/*
// createRemoteProvider - create a new in-memory credential provider
func createRemoteProvider(users []ProxyUser) server.CredentialProvider {
	remoteProvider := server.NewInMemoryProvider()
	for _, user := range users {
		remoteProvider.AddUser(user.Username, user.Password)
	}
	return remoteProvider
}
*/

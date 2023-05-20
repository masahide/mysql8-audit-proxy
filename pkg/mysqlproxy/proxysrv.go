package mysqlproxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/generatepem"
)

// ProxySrv is  the main struct for the proxy server
type ProxySrv struct {
	Config       *ProxyCfg
	listenSock   net.Listener
	tlsConf      *tls.Config
	serverPems   generatepem.Pems
	credProvider server.CredentialProvider
}

// ProxySrv methods
// createListener creates a listening socket
func (p *ProxySrv) createListener() error {
	listenAddr := net.JoinHostPort(p.Config.ProxyListenAddr, p.Config.ProxyListenPort)
	listener, err := net.Listen(p.Config.ProxyListentNet, listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listening socket: %v", err)
	}
	p.listenSock = listener
	log.Printf("Proxy server listening on %s:%s\n", p.Config.ProxyListenAddr, p.Config.ProxyListenPort)

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
	p.credProvider = createRemoteProvider(p.Config.ProxyUsers)
	return nil
}
func (p *ProxySrv) acceptClntConn() {
	for {
		conn, err := p.listenSock.Accept()
		if err != nil {
			fmt.Printf("error accepting client connection: %v\n", err)
			continue
		}
		fmt.Printf("Accepted connection from %s\n", conn.RemoteAddr().String())
		// Handle client connection (e.g., create a new ClntSess and process the connection)
		go p.sessionWorker(conn)
	}
}

func (p *ProxySrv) sessionWorker(c net.Conn) {
	svr := server.NewServer(
		"8.0.12",
		mysql.DEFAULT_COLLATION_ID,
		mysql.AUTH_CACHING_SHA2_PASSWORD,
		[]byte(p.serverPems.Public), p.tlsConf,
	)
	conn, err := server.NewCustomizedConn(c, svr, p.credProvider, server.EmptyHandler{})
	if err != nil {
		log.Printf("Connection error: %v", err)
		return
	}
	user := conn.GetUser()
	log.Printf("user: %s", user)
	sess := &ClientSess{
		ClientMysql: conn,
		ProxySrv:    p,
	}
	err = sess.ConnectToMySQL()
	if err != nil {
		log.Printf("connect to mysql  error: %v", err)
	}

}

// createRemoteProvider - create a new in-memory credential provider
func createRemoteProvider(users []ProxyUser) server.CredentialProvider {
	remoteProvider := server.NewInMemoryProvider()
	for _, user := range users {
		remoteProvider.AddUser(user.Username, user.Password)
	}
	return remoteProvider
}
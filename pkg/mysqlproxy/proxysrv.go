package mysqlproxy

import (
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
	// p.credProvider = createRemoteProvider(p.Config.ProxyUsers)
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
	targetUser, targetAddr, targetPasswrd := getTargetInfo(user)
	if len(targetPasswrd) == 0 {
		targetPasswrd, _, _ = p.credProvider.GetCredential(user)
	}
	sess := &ClientSess{
		ClientMysql:    conn,
		TargetNet:      "TCP",
		TargetAddr:     targetAddr,
		TargetUser:     targetUser,
		TargetPassword: targetPasswrd,
		ProxySrv:       p,
	}
	err = sess.ConnectToMySQL()
	if err != nil {
		log.Printf("connect to mysql  error: %v", err)
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

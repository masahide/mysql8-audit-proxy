package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/generatepem"
)

type RemoteThrottleProvider struct {
	*server.InMemoryProvider
	delay int // in milliseconds
}

func (m *RemoteThrottleProvider) GetCredential(username string) (password string, found bool, err error) {
	time.Sleep(time.Millisecond * time.Duration(m.delay))
	return m.InMemoryProvider.GetCredential(username)
}

type Specification struct {
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:":3306"`
	Debug         bool
	Port          int
	User          string
	Users         []string
	Rate          float32
	Timeout       time.Duration
	ColorCodes    map[string]int
}

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
	c := generatepem.Config{}
	os.Setenv("HOST", "localhost")
	if err := envconfig.Process("", &c); err != nil {
		log.Fatal(err)
	}
	caPems, serverPems, err := generatepem.Generate(c)
	if err != nil {
		log.Fatal(err)
	}
	// user either the in-memory credential provider or the remote credential provider (you can implement your own)
	//inMemProvider := server.NewInMemoryProvider()
	//inMemProvider.AddUser("root", "123")
	remoteProvider := &RemoteThrottleProvider{server.NewInMemoryProvider(), 10 + 50}
	remoteProvider.AddUser("root", "123")
	var tlsConf = server.NewServerTLSConfig(
		[]byte(caPems.Cert),
		[]byte(serverPems.Cert),
		[]byte(serverPems.Key),
		tls.VerifyClientCertIfGiven,
	)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			// Create a connection with user root and an empty password.
			// You can use your own handler to handle command here.
			svr := server.NewServer(
				"8.0.12",
				mysql.DEFAULT_COLLATION_ID,
				mysql.AUTH_CACHING_SHA2_PASSWORD,
				[]byte(serverPems.Public), tlsConf,
			)
			//conn, err := server.NewConn(c, "root", "fugga", server.EmptyHandler{})
			conn, err := server.NewCustomizedConn(c, svr, remoteProvider, server.EmptyHandler{})

			if err != nil {
				log.Printf("Connection error: %v", err)
				return
			}
			user := conn.GetUser()
			log.Printf("user: %s", user)

			for {
				err = conn.HandleCommand()
				if err != nil {
					log.Printf(`Could not handle command: %v`, err)
					break
				}
			}
		}()
	}
}

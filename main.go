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
	"github.com/masahide/mysql8-audit-proxy/pkg/serverconfig"
)

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
	remoteProvider := server.NewInMemoryProvider()
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
			dir, err := os.UserConfigDir()
			if err != nil {
				log.Fatal(err)
			}
			chandler := serverconfig.NewConfigHandler(dir)
			conn, err := server.NewCustomizedConn(c, svr, remoteProvider, chandler)

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

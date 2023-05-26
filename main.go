package main

import (
	"crypto/tls"
	"fmt"
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
			testHandler := &testHandler{
				Manager: serverconfig.NewManager(dir),
			}
			conn, err := server.NewCustomizedConn(c, svr, remoteProvider, testHandler)

			if err != nil {
				log.Printf("Connection error: %v", err)
				return
			}
			user := conn.GetUser()
			log.Printf("user: %s", user)

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

type testHandler struct {
	server.EmptyHandler
	*serverconfig.Manager
	err error
	res *mysql.Result
}

func (h *testHandler) UseDB(dbName string) error {
	return nil
}

type serverConfig struct {
	Servers map[string]Server
}
type Server struct {
	ProxyUser    string
	Password     string
	host         string
	Port         string
	User         string
	HostPassword string
}

func (h *testHandler) selectStmt(p *serverconfig.ParsedQuery) {
	h.res = &mysql.Result{}
	col, data, err := h.Select(p)
	if err != nil {
		h.err = err
		return
	}
	r, _ := mysql.BuildSimpleResultset(col, data, false)
	h.res = &mysql.Result{Resultset: r}
}
func (h *testHandler) insertStmt(p *serverconfig.ParsedQuery) {
	var n uint64
	n, h.err = h.Insert(p)
	h.res = &mysql.Result{AffectedRows: n}

}
func (h *testHandler) updateStmt(p *serverconfig.ParsedQuery) {
	var n uint64
	n, h.err = h.Update(p)
	h.res = &mysql.Result{AffectedRows: n}
}
func (h *testHandler) deleteStmt(p *serverconfig.ParsedQuery) {
	var n uint64
	n, h.err = h.Delete(p)
	h.res = &mysql.Result{AffectedRows: n}
}

func (h *testHandler) handleQuery(query string, binary bool) (*mysql.Result, error) {
	astNode, err := serverconfig.Parse(query)
	if err != nil {
		return nil, err
	}
	data := serverconfig.NewParsedQuery(
		map[string]func(p *serverconfig.ParsedQuery){
			serverconfig.SelectStmt: h.selectStmt,
			serverconfig.InsertStmt: h.insertStmt,
			serverconfig.UpdateStmt: h.updateStmt,
			serverconfig.DeleteStmt: h.deleteStmt,
		},
	)
	(*astNode).Accept(data)
	return h.res, h.err
}

func (h *testHandler) HandleQuery(query string) (*mysql.Result, error) {
	return h.handleQuery(query, false)
}

func (h *testHandler) HandleOtherCommand(cmd byte, data []byte) error {
	return mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d is not supported now", cmd))
}

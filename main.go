package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
	"github.com/kelseyhightower/envconfig"
	"github.com/masahide/mysql8-audit-proxy/pkg/generatepem"
	"github.com/pingcap/errors"
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
			testHandler := &testHandler{}
			conn, err := server.NewCustomizedConn(c, svr, remoteProvider, testHandler)

			if err != nil {
				log.Printf("Connection error: %v", err)
				return
			}
			user := conn.GetUser()
			log.Printf("user: %s", user)

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
			/*
				for _, row := range r.Values {
					for _, val := range row {
						v := val.Value() // interface{}
						log.Printf("value: %v", v)
					}
				}
			*/

			db := clientConn.GetDB()
			log.Printf("client DB:%s", db)
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

func getConfig() serverConfig {
	conf := serverConfig{
		Servers: map[string]Server{
			"user1@localhost": {ProxyUser: "user1", Password: "123", host: "localhost", Port: "3306", User: "root", HostPassword: ""},
			"user2@localhost": {ProxyUser: "user2", Password: "123", host: "localhost", Port: "3306", User: "root", HostPassword: ""},
		},
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "mysql8-audit-proxy.json"))
	if err != nil {
		return conf
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&conf); err != nil {
		log.Printf("error decoding json: %v file:%s. using default empty config", err, f.Name())
		return conf
	}
	return conf
}

func putConfig(conf serverConfig) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "mysql8-audit-proxy.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conf)
}

func (h *testHandler) handleQuery(query string, binary bool) (*mysql.Result, error) {
	ss := strings.Split(query, " ")
	switch strings.ToLower(ss[0]) {
	case "select":
		var r *mysql.Resultset
		var err error
		//for handle go mysql driver select @@max_allowed_packet
		if strings.Contains(strings.ToLower(query), "max_allowed_packet") {
			r, err = mysql.BuildSimpleResultset([]string{"@@max_allowed_packet"}, [][]interface{}{
				{mysql.MaxPayloadLen},
			}, binary)
		} else {
			conf := getConfig()
			r, err = mysql.BuildSimpleResultset(
				[]string{"ProxyUser", "Password", "Host", "Port", "User", "HostPassword"},
				func() [][]interface{} {
					ss := make([][]interface{}, 0, len(conf.Servers))
					for _, s := range conf.Servers {
						ss = append(ss, []interface{}{s.ProxyUser, s.Password, s.host, s.Port, s.User, s.HostPassword})
					}
					return ss
				}(), binary)
		}

		if err != nil {
			return nil, errors.Trace(err)
		} else {
			return &mysql.Result{
				Status:       0,
				Warnings:     0,
				InsertId:     0,
				AffectedRows: 0,
				Resultset:    r,
			}, nil
		}
	case "insert":
		return &mysql.Result{
			Status:       0,
			Warnings:     0,
			InsertId:     1,
			AffectedRows: 0,
			Resultset:    nil,
		}, nil
	case "delete", "update", "replace":
		return &mysql.Result{
			Status:       0,
			Warnings:     0,
			InsertId:     0,
			AffectedRows: 1,
			Resultset:    nil,
		}, nil
	default:
		return nil, fmt.Errorf("invalid query %s", query)
	}
}

func (h *testHandler) HandleQuery(query string) (*mysql.Result, error) {
	return h.handleQuery(query, false)
}

func (h *testHandler) HandleOtherCommand(cmd byte, data []byte) error {
	return mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d is not supported now", cmd))
}

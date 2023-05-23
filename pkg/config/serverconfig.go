package serverconfig

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	appName       = "mysql8-audit-proxy"
	appConfigDir  = "." + appName
	appConfigfile = "config.json"
)

type Manager struct {
	configDir   string
	serverIndex map[string]int
}
type Config struct {
	Servers []Server
}

type Server struct {
	ProxyUser    string
	Password     string
	Host         string
	Port         string
	User         string
	HostPassword string
}

var (
	defaultConfig = Config{
		Servers: []Server{
			{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
			{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
		},
	}
)

func NewManager(dir string) *Manager {
	return &Manager{
		configDir:   dir,
		serverIndex: map[string]int{},
	}
}

func NewConfig() *Config {
	c := &defaultConfig
	return c
}

func (m *Manager) makeIndex(c *Config) {
	m.serverIndex = map[string]int{}
	for i := range c.Servers {
		m.serverIndex[c.Servers[i].ProxyUser] = i
	}
}

func (m *Manager) GetConfig() *Config {
	c := NewConfig()
	m.makeIndex(c)
	f, err := os.Open(filepath.Join(m.configDir, appConfigDir, appConfigfile))
	if err != nil {
		return c
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(c); err != nil {
		log.Printf("error decoding json: %v file:%s. using default empty config", err, f.Name())
	}
	m.makeIndex(c)
	return c
}

func (m *Manager) GetServer(proxyUser string, s []Server) *Server {
	if i, ok := m.serverIndex[proxyUser]; ok {
		return &s[i]
	}
	return nil
}

func (m *Manager) PutConfig(conf *Config) error {
	if err := os.MkdirAll(filepath.Join(m.configDir, appConfigDir), 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(m.configDir, appConfigDir, appConfigfile))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(conf)
}

func (m *Manager) deleteConfig() error {
	return os.RemoveAll(filepath.Join(m.configDir, appConfigDir))
}

func (m *Manager) Insert(p *ParsedQuery) error {
	conf := m.GetConfig()
	err := m.insert(p, conf)
	if err != nil {
		return err
	}
	return m.PutConfig(conf)
}

func (m *Manager) insert(p *ParsedQuery, conf *Config) error {
	servers, err := columnsToConfig(p)
	if err != nil {
		return err
	}
	for _, server := range servers {
		if _, ok := m.serverIndex[server.ProxyUser]; ok {
			return fmt.Errorf("allready exists proxyUser:%s", server.ProxyUser)
		}
		conf.Servers = append(conf.Servers, server)
		m.serverIndex[server.ProxyUser] = len(conf.Servers) - 1
	}
	return nil
}

func (m *Manager) update(p *ParsedQuery, conf *Config) error {
	updateData, err := columnsToConfig(p)
	if err != nil {
		return err
	}
	for _, u := range updateData {
		i := m.serverIndex[u.ProxyUser]
		if conf.Servers[i], err = upsateColumns(p, u); err != nil {
			return err
		}
	}

	return m.PutConfig(conf)
}

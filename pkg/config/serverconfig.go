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
	configDir string
}
type Config struct {
	Servers map[string]Server
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
		Servers: map[string]Server{
			"admin":           {ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
			"user1@localhost": {ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
		},
	}
)

func NewManager(dir string) *Manager {
	return &Manager{configDir: dir}
}

func (m *Manager) GetConfig() Config {
	f, err := os.Open(filepath.Join(m.configDir, appConfigDir, appConfigfile))
	if err != nil {
		return defaultConfig
	}
	conf := Config{}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&conf); err != nil {
		log.Printf("error decoding json: %v file:%s. using default empty config", err, f.Name())
		return conf
	}
	return conf
}

func (m *Manager) PutConfig(conf Config) error {
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

func (m *Manager) insert(p *ParsedQuery) error {
	conf := m.GetConfig()
	servers, err := columnsToConfig(p)
	if err != nil {
		return err
	}
	for _, server := range servers {
		if _, ok := conf.Servers[server.ProxyUser]; ok {
			return fmt.Errorf("allready exists proxyUser:%s", server.ProxyUser)
		}
		conf.Servers[server.ProxyUser] = server
	}
	return m.PutConfig(conf)
}

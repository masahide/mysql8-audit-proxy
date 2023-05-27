package serverconfig

import (
	"encoding/json"
	"errors"
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

type Server struct {
	User     string
	Password string
}

var (
	defaultConfig = Config{
		Servers: []Server{
			{User: "admin", Password: "pass"},
		},
	}
)

type Config struct {
	Servers []Server
}

func NewConfig() *Config {
	c := &defaultConfig
	return c
}

func NewManager(dir string) *Manager {
	return &Manager{
		configDir:   dir,
		serverIndex: map[string]int{},
	}
}

func (m *Manager) makeIndex(c *Config) {
	m.serverIndex = map[string]int{}
	for i := range c.Servers {
		m.serverIndex[c.Servers[i].User] = i
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

func (m *Manager) Insert(p *ParsedQuery) (uint64, error) {
	conf := m.GetConfig()
	n, err := m.insert(p, conf)
	if err != nil {
		return n, err
	}
	return n, m.PutConfig(conf)
}

func (m *Manager) insert(p *ParsedQuery, conf *Config) (uint64, error) {
	servers, err := columnsToConfig(p)
	n := uint64(0)
	if err != nil {
		return n, err
	}
	for _, server := range servers {
		if _, ok := m.serverIndex[server.User]; ok {
			return n, fmt.Errorf("allready exists proxyUser:%s", server.User)
		}
		conf.Servers = append(conf.Servers, server)
		m.serverIndex[server.User] = len(conf.Servers) - 1
		n++
	}
	return n, nil
}

func (m *Manager) Select(p *ParsedQuery) ([]string, [][]interface{}, error) {
	conf := m.GetConfig()
	rows, err := whereColumnsToConfig(p, conf.Servers)
	if err != nil {
		return nil, nil, err
	}
	return selectResultset(p, rows)
}
func (m *Manager) Update(p *ParsedQuery) (uint64, error) {
	conf := m.GetConfig()
	n, err := m.update(p, conf)
	if err != nil {
		return n, err
	}
	return n, m.PutConfig(conf)
}

func (m *Manager) update(p *ParsedQuery, conf *Config) (uint64, error) {
	n := uint64(0)
	rows, err := whereColumnsToConfig(p, conf.Servers)
	if err != nil {
		return n, err
	}
	if len(rows) == 0 {
		return n, errors.New("no update data")
	}
	for _, u := range rows {
		i, ok := m.serverIndex[u.User]
		if !ok {
			return n, fmt.Errorf("proxyUser:%s not found", u.User)
		}
		if conf.Servers[i], err = updateColumns(p, u); err != nil {
			return n, err
		}
		n++
	}

	return n, err
}

func (m *Manager) Delete(p *ParsedQuery) (uint64, error) {
	conf := m.GetConfig()
	n, err := m.delete(p, conf)
	if err != nil {
		return n, err
	}
	return n, m.PutConfig(conf)
}
func (m *Manager) delete(p *ParsedQuery, conf *Config) (uint64, error) {
	n := uint64(0)
	res := make([]Server, 0, len(conf.Servers))
	rows, err := whereColumnsToConfig(p, conf.Servers)
	if err != nil {
		return n, err
	}
	if len(rows) == 0 {
		return n, errors.New("not found data")
	}
	for _, s := range conf.Servers {
		if !func() bool {
			for _, u := range rows {
				if s.User == u.User {
					return true
				}
			}
			return false
		}() {
			res = append(res, s)
			continue
		}
		n++
	}
	conf.Servers = res
	return n, err
}

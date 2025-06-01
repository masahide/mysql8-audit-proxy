package serverconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

const (
	appName       = "mysql8-audit-proxy"
	appConfigDir  = appName
	appConfigfile = "config.json"
)

type Manager struct {
	configDir   string
	filePath    string
	serverIndex map[string]int
	mu          sync.RWMutex
}

type Server struct {
	User     string
	Password string
}

var (
	defaultConfig = func(key []byte) *Config {
		return &Config{
			Key: key,
			Servers: []Server{
				{User: "admin", Password: mustEncrypt(key, "pass")},
			},
		}
	}
)

type Config struct {
	Servers []Server
	Key     []byte
}

func NewConfig() *Config {
	key, err := generateKey()
	if err != nil {
		log.Fatal(err)
	}
	return defaultConfig(key)
}

func NewManager(dir string) *Manager {
	return &Manager{
		configDir:   dir,
		filePath:    filepath.Join(dir, appConfigDir, appConfigfile),
		serverIndex: map[string]int{},
	}
}

func (m *Manager) PrintPathInfo() string {
	b, err := json.Marshal(map[string]string{"filePath": m.filePath})
	if err != nil {
		return ""
	}
	return string(b)
}
func (m *Manager) makeIndex(conf *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverIndex = map[string]int{}
	for i := range conf.Servers {
		m.serverIndex[conf.Servers[i].User] = i
	}
}

func (m *Manager) GetConfig() *Config {
	conf := NewConfig()
	// b, _ := json.MarshalIndent(conf, "", "  ")
	// log.Print(string(b))
	m.makeIndex(conf)
	f, err := os.Open(m.filePath)
	if err != nil {
		return conf
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(conf); err != nil {
		log.Printf("error decoding json: %v file:%s. using default empty config", err, f.Name())
	}
	if conf.Key == nil || len(conf.Key) == 0 {
		conf.Key, err = generateKey()
		if err != nil {
			log.Fatal(err)
		}
	}
	m.makeIndex(conf)
	//log.Printf("loaded config: %# v", conf)
	return conf
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, server := range servers {
		if _, ok := m.serverIndex[server.User]; ok {
			return n, fmt.Errorf("allready exists proxyUser:%s", server.User)
		}
		server.Password, err = encrypt(conf.Key, server.Password)
		if err != nil {
			return n, err
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range rows {
		i, ok := m.serverIndex[u.User]
		if !ok {
			return n, fmt.Errorf("proxyUser:%s not found", u.User)
		}
		s, err := updateColumns(p, u)
		if err != nil {
			return n, err
		}
		s.Password, err = encrypt(conf.Key, s.Password)
		if err != nil {
			return n, err
		}
		conf.Servers[i] = s
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

type serverInfo struct {
	Addr     string
	User     string
	Password []byte
}

const defaultMySQLPort = "3306"

/*
func getServerInfo(input string) (serverInfo, error) {
	var info serverInfo
	splitAt := strings.Split(input, "@")
	switch len(splitAt) {
	case 1:
		info.User = splitAt[0]
	case 2:
		info.Addr = splitAt[1]
		splitColon := strings.Split(splitAt[0], ":")
		switch len(splitColon) {
		case 1:
			info.User = splitColon[0]
			case 2:
				info.User = splitColon[0]
				info.Password = splitColon[1]
		default:
			return serverInfo{}, errors.New("invalid input")
		}
	default:
		return serverInfo{}, errors.New("invalid input")
	}

	if info.Addr != "" {
		splitColon := strings.Split(info.Addr, ":")
		if len(splitColon) == 1 {
			info.Addr = info.Addr + ":" + defaultMySQLPort
		}
	}

	return info, nil
}
*/

func (m *Manager) getServer(conf *Config, username string) *Server {
	for _, s := range conf.Servers {
		re, err := regexp.Compile(`^` + s.User + `$`)
		if err != nil {
			if s.User == username {
				return &s
			}
			continue
		}
		if re.MatchString((username)) {
			return &s
		}
	}
	return nil
	/*
		svi, err := getServerInfo(username)
		if err != nil {
			return nil
		}

		return &Server{User: svi.User, Password: svi.Password}
	*/
}

func (m *Manager) GetPassword(username string) (string, error) {
	conf := m.GetConfig()
	s := m.getServer(conf, username)
	if s == nil {
		return "", errors.New("not found")
	}
	p, err := decrypt(conf.Key, s.Password)
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func generateKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func encrypt(key []byte, data string) (string, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
	s := base64.RawStdEncoding.EncodeToString(ciphertext)
	return s, nil
}

func mustEncrypt(key []byte, data string) string {
	crypt, err := encrypt(key, data)
	if err != nil {
		log.Fatal(err)
	}
	return crypt
}

func decrypt(key []byte, data string) (string, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", err
	}
	b, err := base64.RawStdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	nonce, ciphertext := b[:gcm.NonceSize()], b[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

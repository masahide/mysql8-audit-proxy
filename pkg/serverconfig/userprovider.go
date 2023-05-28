package serverconfig

import (
	"regexp"
)

// interface for user credential provider
// hint: can be extended for more functionality
// =================================IMPORTANT NOTE===============================
// if the password in a third-party credential provider could be updated at runtime, we have to invalidate the caching
// for 'caching_sha2_password' by calling 'func (s *Server)InvalidateCache(string, string)'.
/*
type CredentialProvider interface {
	// check if the user exists
	CheckUsername(username string) (bool, error)
	// get user credential
	GetCredential(username string) (password string, found bool, err error)
}
*/

func NewConfigProvider(m *Manager) *ConfigProvider {
	return &ConfigProvider{Manager: m}
}

// implements a in memory credential provider
type ConfigProvider struct {
	*Manager
	//mu      sync.Mutex
	//servers []Server
}

func (m *ConfigProvider) getServer(username string) *Server {
	c := m.GetConfig()
	for _, s := range c.Servers {
		//log.Printf("s:%# v",s)
		re, err := regexp.Compile(s.User)
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
}
func (m *ConfigProvider) CheckUsername(username string) (found bool, err error) {
	return m.getServer(username) != nil, nil
}

func (m *ConfigProvider) GetCredential(username string) (password string, found bool, err error) {
	s := m.getServer(username)
	if s == nil {
		return "", false, nil
	}
	return s.Password, true, nil
}

type Provider ConfigProvider

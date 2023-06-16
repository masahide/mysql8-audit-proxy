package mysqlproxy

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

type userManager interface {
	GetPassword(username string) (string, error)
}

func NewConfigProvider(m userManager) *ConfigProvider {
	return &ConfigProvider{userManager: m}
}

// implements a in memory credential provider
type ConfigProvider struct {
	userManager
	//mu      sync.Mutex
	//servers []Server
}

func (m *ConfigProvider) CheckUsername(username string) (bool, error) {
	//log.Printf("CheckUsername username:%s", username)
	_, err := m.GetPassword(username)
	return err == nil, err
}

func (m *ConfigProvider) GetCredential(username string) (password string, found bool, err error) {
	//log.Printf("GetCredential username:%s", username)
	pw, err := m.GetPassword(username)
	if err != nil {
		return "", false, nil
	}
	return pw, true, nil
}

type Provider ConfigProvider

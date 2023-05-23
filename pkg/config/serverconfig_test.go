package serverconfig

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManager(t *testing.T) {
	testCases := []struct {
		name     string
		dir      string
		config   Config
		expected *Config
	}{
		{
			name: "default config",
			dir:  "default",
			config: Config{
				Servers: []Server{
					{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
					{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				},
			},
			expected: NewConfig(),
		},
		{
			name: "custom config",
			dir:  "custom",
			config: Config{
				Servers: []Server{
					{ProxyUser: "custom", Password: "custom", Host: "custom", Port: "custom", User: "custom", HostPassword: "custom"},
				},
			},
			expected: &Config{
				Servers: []Server{
					{ProxyUser: "custom", Password: "custom", Host: "custom", Port: "custom", User: "custom", HostPassword: "custom"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager(tc.dir)

			t.Run("putConfig", func(t *testing.T) {
				// Test putConfig
				err := m.PutConfig(&tc.config)
				if err != nil {
					t.Fatalf("PutConfig error: %v", err)
				}
			})

			t.Run("GetConfig", func(t *testing.T) {
				// Test GetConfig
				got := m.GetConfig()
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("mismatch (-got +expected):\n%s", diff)
				}
			})

			t.Run("deleteConfig", func(t *testing.T) {
				// Test deleteConfig
				err := m.deleteConfig()
				if err != nil {
					t.Fatalf("deleteConfig error: %v", err)
				}
			})

			t.Run("GetConfigAfterDelete", func(t *testing.T) {
				// Test GetConfig after deleteConfig
				got := m.GetConfig()
				if diff := cmp.Diff(*got, defaultConfig); diff != "" {
					t.Errorf("GetConfig after deleteConfig mismatch (-got +defaultConfig):\n%s", diff)
				}
			})
		})
	}
}

func TestManager_Insert(t *testing.T) {
	testCases := []struct {
		name            string
		parsedQuery     ParsedQuery
		initialConfig   *Config
		expectedErr     error
		expectedServers []Server
	}{
		{
			name: "insert new server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{"ProxyUser", "Password", "Host", "Port", "User", "HostPassword"},
					Values:  []string{"user2", "password2", "example.com", "5432", "user2", "pass2"},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
					{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				},
			},
			expectedErr: nil,
			expectedServers: []Server{
				{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
				{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				{ProxyUser: "user2", Password: "password2", Host: "example.com", Port: "5432", User: "user2", HostPassword: "pass2"},
			},
		},
		{
			name: "insert existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{"ProxyUser", "Password", "Host", "Port", "User", "HostPassword"},
					Values:  []string{"admin", "newpass", "", "", "", ""},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
					{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				},
			},
			expectedErr: fmt.Errorf("allready exists proxyUser:admin"),
		},
		{
			name: "insert with missing columns",
			parsedQuery: ParsedQuery{
				Query: Query{
					Values: []string{"user3", "password3", "example.com", "5432", "user3", "pass3"},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
					{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				},
			},
			expectedErr: nil,
			expectedServers: []Server{
				{ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
				{ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				{ProxyUser: "user3", Password: "password3", Host: "example.com", Port: "5432", User: "user3", HostPassword: "pass3"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				configDir:   "config",
				serverIndex: make(map[string]int),
			}
			m.makeIndex(tc.initialConfig)
			err := m.insert(&tc.parsedQuery, tc.initialConfig)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if tc.expectedErr != nil {
				t.Errorf("expected error: %v, but got nil", tc.expectedErr)
			} else {
				if diff := cmp.Diff(tc.initialConfig.Servers, tc.expectedServers); diff != "" {
					t.Errorf("mismatch (-got +expected):\n%s", diff)
				}
			}
		})
	}
}

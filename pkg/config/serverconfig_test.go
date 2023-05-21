package serverconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestManager(t *testing.T) {
	testCases := []struct {
		name     string
		dir      string
		config   Config
		expected Config
	}{
		{
			name: "default config",
			dir:  "default",
			config: Config{
				Servers: map[string]Server{
					"admin":           {ProxyUser: "admin", Password: "pass", Host: "", Port: "", User: "", HostPassword: ""},
					"user1@localhost": {ProxyUser: "user1@localhost", Password: "123", Host: "localhost", Port: "3306", User: "root", HostPassword: ""},
				},
			},
			expected: defaultConfig,
		},
		{
			name: "custom config",
			dir:  "custom",
			config: Config{
				Servers: map[string]Server{
					"custom": {ProxyUser: "custom", Password: "custom", Host: "custom", Port: "custom", User: "custom", HostPassword: "custom"},
				},
			},
			expected: Config{
				Servers: map[string]Server{
					"custom": {ProxyUser: "custom", Password: "custom", Host: "custom", Port: "custom", User: "custom", HostPassword: "custom"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager(tc.dir)

			t.Run("putConfig", func(t *testing.T) {
				// Test putConfig
				err := m.PutConfig(tc.config)
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
				if diff := cmp.Diff(got, defaultConfig); diff != "" {
					t.Errorf("GetConfig after deleteConfig mismatch (-got +defaultConfig):\n%s", diff)
				}
			})
		})
	}
}

package serverconfig

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pingcap/tidb/parser/opcode"
)

func TestManager(t *testing.T) {
	dir, err := os.MkdirTemp("", "mysqlaudit-proxy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	testCases := []struct {
		name     string
		dir      string
		config   Config
		user     string
		pass     string
		expected *Config
	}{
		{
			name: "default config",
			dir:  dir,
			config: Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			user: "admin",
			pass: "pass",
			expected: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
		},
		{
			name: "custom config",
			dir:  dir,
			config: Config{
				Servers: []Server{
					{User: "custom", Password: "pass1"},
					{User: "custom2", Password: "pass2"},
				},
			},
			user: "custom2",
			pass: "pass2",
			expected: &Config{
				Servers: []Server{
					{User: "custom", Password: "pass1"},
					{User: "custom2", Password: "pass2"},
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

			t.Run("GetPassword", func(t *testing.T) {
				got, _ := m.GetPassword(tc.user)
				if got != tc.pass {
					t.Errorf("mismatch (-got +expected):-%s,+%s", got, tc.pass)
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
		expectN         uint64
	}{
		{
			name: "insert new server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{"User", "Password"},
					Values:  []string{"user2", "password2"},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: nil,
			expectedServers: []Server{
				{User: "admin", Password: "pass"},
				{User: "user1@localhost", Password: "123"},
				{User: "user2", Password: "password2"},
			},
			expectN: 1,
		},
		{
			name: "insert existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{"User", "Password"},
					Values:  []string{"admin", "newpass"},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: fmt.Errorf("allready exists proxyUser:admin"),
			expectN:     0,
		},
		{
			name: "insert with select * columns",
			parsedQuery: ParsedQuery{
				Query: Query{
					Values: []string{"user3", "password3"},
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: nil,
			expectedServers: []Server{
				{User: "admin", Password: "pass"},
				{User: "user1@localhost", Password: "123"},
				{User: "user3", Password: "password3"},
			},
			expectN: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				configDir:   "config",
				serverIndex: make(map[string]int),
			}
			m.makeIndex(tc.initialConfig)
			n, err := m.insert(&tc.parsedQuery, tc.initialConfig)
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
			if n != tc.expectN {
				t.Errorf("n  mismatch: got %d, want %d", n, tc.expectN)
			}
		})
	}
}

func TestManager_Update(t *testing.T) {
	testCases := []struct {
		name          string
		parsedQuery   ParsedQuery
		initialConfig *Config
		expectedErr   error
		expectN       uint64
	}{
		{
			name: "update existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{Password, User},
					Values:       []string{"newpass", "example.com"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.EQ,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: nil,
			expectN:     1,
		},
		{
			name: "update non-existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{Password, User},
					Values:       []string{"newpass", "example.com"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"nonexisting"},
					WhereOp:      opcode.EQ,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: fmt.Errorf("no update data"),
			expectN:     0,
		},
		{
			name: "update with invalid where operation",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{Password},
					Values:       []string{"newpass"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.LT,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: fmt.Errorf("where only supports equal operation"),
			expectN:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				configDir:   "config",
				serverIndex: make(map[string]int),
			}
			m.makeIndex(tc.initialConfig)
			n, err := m.update(&tc.parsedQuery, tc.initialConfig)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if tc.expectedErr != nil {
				t.Errorf("expected error: %v, but got nil", tc.expectedErr)
			} else if n != tc.expectN {
				t.Errorf("n  mismatch: got %d, want %d", n, tc.expectN)
			}
		})
	}
}

func TestManager_Delete(t *testing.T) {
	testCases := []struct {
		name            string
		parsedQuery     ParsedQuery
		initialConfig   *Config
		expectedErr     error
		expectedServers []Server
		expectN         uint64
	}{
		{
			name: "update existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{"Password", "User"},
					Values:       []string{"newpass", "example.com"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.EQ,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedServers: []Server{
				{User: "user1@localhost", Password: "123"},
			},
			expectedErr: nil,
			expectN:     1,
		},
		{
			name: "update non-existing server",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{Password, User},
					Values:       []string{"newpass", "example.com"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"nonexisting"},
					WhereOp:      opcode.EQ,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: fmt.Errorf("not found data"),
			expectedServers: []Server{
				{User: "admin", Password: "pass"},
				{User: "user1@localhost", Password: "123"},
			},
			expectN: 2,
		},
		{
			name: "update with invalid where operation",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns:      []string{"Password"},
					Values:       []string{"newpass"},
					WhereColumns: []string{"User"},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.LT,
				},
			},
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user1@localhost", Password: "123"},
				},
			},
			expectedErr: fmt.Errorf("where only supports equal operation"),
			expectN:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				configDir:   "config",
				serverIndex: make(map[string]int),
			}
			m.makeIndex(tc.initialConfig)
			n, err := m.delete(&tc.parsedQuery, tc.initialConfig)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if tc.expectedErr != nil {
				t.Errorf("expected error: %v, but got nil", tc.expectedErr)
			} else if n != tc.expectN {
				t.Errorf("n  mismatch: got %d, want %d", n, tc.expectN)
			} else if diff := cmp.Diff(tc.initialConfig.Servers, tc.expectedServers); diff != "" {
				t.Errorf("mismatch (-got +expected):\n%s", diff)
			}
		})
	}
}

func TestGetServerInfo(t *testing.T) {
	testCases := []struct {
		input       string
		expected    serverInfo
		expectedErr error
	}{
		{
			input: "user1@server:3307",
			expected: serverInfo{
				Addr:     "server:3307",
				User:     "user1",
				Password: "",
			},
			expectedErr: nil,
		},
		{
			input: "user2:pass@server:3307",
			expected: serverInfo{
				Addr:     "server:3307",
				User:     "user2",
				Password: "pass",
			},
			expectedErr: nil,
		},
		{
			input: "user3@server",
			expected: serverInfo{
				Addr:     "server" + ":" + defaultMySQLPort,
				User:     "user3",
				Password: "",
			},
			expectedErr: nil,
		},
		{
			input: "admin4",
			expected: serverInfo{
				Addr:     "",
				User:     "admin4",
				Password: "",
			},
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := getServerInfo(tc.input)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if tc.expectedErr != nil {
				t.Errorf("expected error: %v, but got nil", tc.expectedErr)
			} else if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("mismatch (-got +expected):\n%s", diff)
			}
		})
	}
}

/*
func TestGetServer(t *testing.T) {
	testCases := []struct {
		input         string
		initialConfig *Config
		expected      string
		expectedErr   error
	}{
		{
			input: "admin",
			initialConfig: &Config{
				Servers: []Server{
					{User: "admin", Password: "pass"},
					{User: "user", Password: "123"},
				},
			},
			expected:    "pass",
			expectedErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			m := &Manager{
				configDir:   "config",
				serverIndex: make(map[string]int),
			}
			m.makeIndex(tc.initialConfig)
			got, err := m.GetPassword(tc.input)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if tc.expectedErr != nil {
				t.Errorf("expected error: %v, but got nil", tc.expectedErr)
			} else if got != tc.expected {
				t.Errorf("mismatch (-got +expected):-%s,+%s", got, tc.expected)
			}
		})
	}
}
*/

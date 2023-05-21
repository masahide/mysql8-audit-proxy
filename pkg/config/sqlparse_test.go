package serverconfig

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pingcap/tidb/parser/opcode"
)

func TestParse(t *testing.T) {
	testcase := []struct {
		name     string
		sql      string
		expected Query
	}{
		{
			name: "insert",
			sql:  "INSERT INTO syain(name1,name2,romaji) VALUES ('bbbbbbb','aaaa','suzuki');",
			expected: Query{
				Statement:    InsertStmt,
				TableName:    "syain",
				Text:         "INSERT INTO syain(name1,name2,romaji) VALUES ('bbbbbbb','aaaa','suzuki');",
				Columns:      []string{"name1", "name2", "romaji"},
				Values:       []string{"bbbbbbb", "aaaa", "suzuki"},
				WhereColumns: []string{},
				WhereValues:  []string{},
			},
		},
		{
			name: "delete",
			sql:  "delete from users where name1 = 'bbbbbbb'",
			expected: Query{
				Statement:    DeleteStmt,
				TableName:    "users",
				Text:         "delete from users where name1 = 'bbbbbbb'",
				Columns:      []string{},
				Values:       []string{},
				WhereColumns: []string{"name1"},
				WhereValues:  []string{"bbbbbbb"},
				WhereOp:      opcode.EQ,
			},
		},
		{name: "update", sql: "UPDATE users SET password = 'bbbbbb' WHERE username = 'user1'",
			expected: Query{
				Statement:    UpdateStmt,
				TableName:    "users",
				Text:         "UPDATE users SET password = 'bbbbbb' WHERE username = 'user1'",
				WhereColumns: []string{"username"},
				WhereValues:  []string{"user1"},
				Columns:      []string{"password"},
				Values:       []string{"bbbbbb"},
				WhereOp:      opcode.EQ,
			},
		},
		{name: "select", sql: "SELECT username, password FROM users where username = 'user1'",
			expected: Query{
				Statement:    SelectStmt,
				TableName:    "users",
				Text:         "SELECT username, password FROM users where username = 'user1'",
				WhereColumns: []string{"username"},
				WhereValues:  []string{"user1"},
				Columns:      []string{"username", "password"},
				Values:       []string{},
				WhereOp:      opcode.EQ,
			},
		},
		{name: "select", sql: "SELECT * FROM users where username = 'user2'",
			expected: Query{
				Statement:    SelectStmt,
				TableName:    "users",
				Text:         "SELECT * FROM users where username = 'user2'",
				Columns:      []string{},
				Values:       []string{},
				WhereColumns: []string{"username"},
				WhereValues:  []string{"user2"},
				WhereOp:      opcode.EQ,
			},
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			astNode, err := parse(tt.sql)
			if err != nil {
				t.Fatalf("parse error: %v\n", err.Error())
			}
			//log.Printf("%# v\n", *astNode)
			f := func(p *ParsedQuery) {
				if diff := cmp.Diff(p.Query, tt.expected); diff != "" {
					t.Errorf("mismatch (-got +expected):\n%s", diff)
				}
			}
			data := NewParsedQuery(
				map[string]func(p *ParsedQuery){SelectStmt: f, InsertStmt: f, UpdateStmt: f, DeleteStmt: f},
			)
			(*astNode).Accept(data)
		})
	}
}

func TestColumnsToConfig(t *testing.T) {
	testcase := []struct {
		name     string
		in       ParsedQuery
		expected []Server
		err      error
	}{
		{
			name: "insert1",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{ProxyUser, Password, Host, Port, User, HostPassword},
					Values:       []string{"XXXX", "root", "000000000", "3306", "", ""},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
		},
		{
			name: "insert2",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{},
					Values:       []string{"XXXX", "root", "000000000", "3306", "", ""},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
		},
		{
			name: "insert3",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{"ProXyUser", Password, Host, Port, User, HostPassword},
					Values:       []string{"XXXX", "root", "000000000", "3306", "", ""},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
		},
		{
			name: "insert4",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{},
					Values:       []string{"XXXX", "root", "000000000", "3306", "", "", "XXXX", "root", "000000000", "3306", "", ""},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
		},
		{
			name: "insert4",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{"aaaa"},
					Values:       []string{"XXXX", "root", "000000000", "3306", "", "", "XXXX", "root", "000000000", "3306", "", ""},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
			err: errors.New("column aaaa not found"),
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			got, err := columnsToConfig(&tt.in)
			if err != nil {
				if err.Error() != tt.err.Error() {
					t.Error(err)
				}
			} else {
				if diff := cmp.Diff(got, tt.expected); diff != "" {
					t.Errorf("mismatch (-got +expected):\n%s", diff)
				}
			}
		})
	}

}

func TestSelectResultset(t *testing.T) {
	testcase := []struct {
		name         string
		query        ParsedQuery
		servers      []Server
		expectedCols []string
		expectedVuls [][]interface{}
		err          error
	}{
		{
			name: "select1",
			query: ParsedQuery{
				Query: Query{
					TableName: "",
					Columns:   []string{ProxyUser, Password, Host, Port, User, HostPassword},
				},
			},
			servers: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
			expectedCols: []string{ProxyUser, Password, Host, Port, User, HostPassword},
			expectedVuls: [][]interface{}{
				{"XXXX", "root", "000000000", "3306", "", ""},
			},
		},
		{
			name: "select2",
			query: ParsedQuery{
				Query: Query{
					TableName: "",
					Columns:   []string{lProxyUser, lHostPassword},
				},
			},
			servers: []Server{
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
				{ProxyUser: "XXXX", Password: "root", Host: "000000000", Port: "3306", User: "", HostPassword: ""},
			},
			expectedCols: []string{ProxyUser, HostPassword},
			expectedVuls: [][]interface{}{
				{"XXXX", ""},
				{"XXXX", ""},
				{"XXXX", ""},
				{"XXXX", ""},
			},
		},
	}

	for _, tt := range testcase {
		t.Run(tt.name, func(t *testing.T) {
			gotCols, gotVuls, err := selectResultset(&tt.query, tt.servers)
			if err != nil {
				if err.Error() != tt.err.Error() {
					t.Error(err)
				}
			} else {
				if diff := cmp.Diff(gotCols, tt.expectedCols); diff != "" {
					t.Errorf("mismatch (-gotCols +expectedCols):\n%s", diff)
				}
				if diff := cmp.Diff(gotVuls, tt.expectedVuls); diff != "" {
					t.Errorf("mismatch (-gotVuls +expectedVuls):\n%s", diff)
				}
			}
		})
	}

}
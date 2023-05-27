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
			astNode, err := Parse(tt.sql)
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
					Columns:      []string{User, Password},
					Values:       []string{"XXXX", "root"},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{User: "XXXX", Password: "root"},
			},
		},
		{
			name: "insert2",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{},
					Values:       []string{"XXXX", "root"},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{User: "XXXX", Password: "root"},
			},
		},
		{
			name: "insert3",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{User, Password},
					Values:       []string{"XXXX", "root"},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{User: "XXXX", Password: "root"},
			},
		},
		{
			name: "insert4",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{},
					Values:       []string{"xxxx", "root", "xxxx", "root"},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{User: "xxxx", Password: "root"},
				{User: "xxxx", Password: "root"},
			},
		},
		{
			name: "insert5",
			in: ParsedQuery{
				Query: Query{
					TableName:    "",
					Columns:      []string{"aaaa"},
					Values:       []string{"xxxx", "root", "yyyy", "root"},
					WhereColumns: []string{},
					WhereValues:  []string{},
				},
			},
			expected: []Server{
				{User: "XXXX", Password: "root"},
				{User: "yyyy", Password: "root"},
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
					Columns:   []string{User, Password},
				},
			},
			servers: []Server{
				{User: "XXXX", Password: "root"},
			},
			expectedCols: []string{User, Password},
			expectedVuls: [][]interface{}{
				{"XXXX", "root"},
			},
		},
		{
			name: "select2",
			query: ParsedQuery{
				Query: Query{
					TableName: "",
					Columns:   []string{lUser, lPassword},
				},
			},
			servers: []Server{
				{User: "XXXX", Password: "root"},
				{User: "XXXX", Password: "root"},
				{User: "XXXX", Password: "root"},
				{User: "XXXX", Password: "root"},
			},
			expectedCols: []string{User, Password},
			expectedVuls: [][]interface{}{
				{"XXXX", "root"},
				{"XXXX", "root"},
				{"XXXX", "root"},
				{"XXXX", "root"},
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
func TestSelectResultset2(t *testing.T) {
	servers := []Server{
		{User: "admin", Password: "pass"},
		{User: "user1", Password: "123456"},
	}

	testCases := []struct {
		name            string
		parsedQuery     ParsedQuery
		expectedColumns []string
		expectedRows    [][]interface{}
		expectedErr     error
	}{
		{
			name: "select User, Password",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{User, Password},
				},
			},
			expectedColumns: []string{User, Password},
			expectedRows: [][]interface{}{
				{"admin", "pass"},
				{"user1", "123456"},
			},
			expectedErr: nil,
		},
		{
			name: "select Password, User",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{Password, User},
				},
			},
			expectedColumns: []string{Password, User},
			expectedRows: [][]interface{}{
				{"pass", "admin"},
				{"123456", "user1"},
			},
			expectedErr: nil,
		},
		{
			name: "select *",
			parsedQuery: ParsedQuery{
				Query: Query{
					Columns: []string{},
				},
			},
			expectedColumns: []string{User, Password},
			expectedRows: [][]interface{}{
				{"admin", "pass"},
				{"user1", "123456"},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			columns, rows, err := selectResultset(&tc.parsedQuery, servers)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else {
				if len(columns) != len(tc.expectedColumns) {
					t.Errorf("columns count mismatch: got %d, want %d", len(columns), len(tc.expectedColumns))
				} else {
					for i := range columns {
						if columns[i] != tc.expectedColumns[i] {
							t.Errorf("column mismatch: got %s, want %s", columns[i], tc.expectedColumns[i])
						}
					}
				}

				if len(rows) != len(tc.expectedRows) {
					t.Errorf("rows count mismatch: got %d, want %d", len(rows), len(tc.expectedRows))
				} else {
					for i := range rows {
						if len(rows[i]) != len(tc.expectedRows[i]) {
							t.Errorf("row %d length mismatch: got %d, want %d", i, len(rows[i]), len(tc.expectedRows[i]))
						} else {
							for j := range rows[i] {
								if rows[i][j] != tc.expectedRows[i][j] {
									t.Errorf("row %d column %d mismatch: got %v, want %v", i, j, rows[i][j], tc.expectedRows[i][j])
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestWhereColumnsToConfig(t *testing.T) {
	servers := []Server{
		{User: "admin", Password: "pass"},
		{User: "user1", Password: "123456"},
	}

	testCases := []struct {
		name          string
		parsedQuery   ParsedQuery
		expectedCount int
		expectedErr   error
	}{
		{
			name: "proxyuser = admin",
			parsedQuery: ParsedQuery{
				Query: Query{
					WhereColumns: []string{User},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.EQ,
				},
			},
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			name: "user = aaaa",
			parsedQuery: ParsedQuery{
				Query: Query{
					WhereColumns: []string{User},
					WhereValues:  []string{"user1"},
					WhereOp:      opcode.EQ,
				},
			},
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			name: "user = user1",
			parsedQuery: ParsedQuery{
				Query: Query{
					WhereColumns: []string{User},
					WhereValues:  []string{"user1"},
					WhereOp:      opcode.EQ,
				},
			},
			expectedCount: 1,
			expectedErr:   nil,
		},
		{
			name: "invalid where operation",
			parsedQuery: ParsedQuery{
				Query: Query{
					WhereColumns: []string{User},
					WhereValues:  []string{"admin"},
					WhereOp:      opcode.LT,
				},
			},
			expectedCount: 0,
			expectedErr:   errors.New("where only supports equal operation"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := whereColumnsToConfig(&tc.parsedQuery, servers)
			if err != nil {
				if tc.expectedErr == nil {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != tc.expectedErr.Error() {
					t.Errorf("error mismatch: got %v, want %v", err, tc.expectedErr)
				}
			} else if len(res) != tc.expectedCount {
				t.Errorf("result count mismatch: got %d, want %d", len(res), tc.expectedCount)
			}
		})
	}
}

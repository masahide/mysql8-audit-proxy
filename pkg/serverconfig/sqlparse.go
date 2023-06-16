package serverconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

const (
	UpdateStmt = "udpate"
	InsertStmt = "insert"
	DeleteStmt = "delete"
	SelectStmt = "select"

	Where = "Where"
)

type Query struct {
	Statement    string    `json:",omitempty"`
	TableName    string    `json:",omitempty"`
	Columns      []string  `json:",omitempty"`
	Values       []string  `json:",omitempty"`
	Text         string    `json:",omitempty"`
	CurrentType  string    `json:",omitempty"`
	WhereColumns []string  `json:",omitempty"`
	WhereValues  []string  `json:",omitempty"`
	WhereOp      opcode.Op `json:",omitempty"`
}

type ParsedQuery struct {
	funcs map[string]func(p *ParsedQuery)
	Query
}

func NewParsedQuery(funcs map[string]func(p *ParsedQuery)) *ParsedQuery {
	return &ParsedQuery{
		Query: Query{
			TableName:    "",
			Columns:      []string{},
			Values:       []string{},
			WhereColumns: []string{},
			WhereValues:  []string{},
		},
		funcs: funcs,
	}
}

func (p *ParsedQuery) Enter(in ast.Node) (ast.Node, bool) {
	switch s := in.(type) {
	case *ast.SelectStmt:
		p.Statement = SelectStmt
		p.Text = s.Text()
	case *ast.InsertStmt:
		p.Statement = InsertStmt
		p.Text = s.Text()
	case *ast.UpdateStmt:
		p.Statement = UpdateStmt
		p.Text = s.Text()
	case *ast.DeleteStmt:
		p.Statement = DeleteStmt
		p.Text = s.Text()
	case *ast.TableName:
		p.TableName = s.Name.L
	case *ast.BinaryOperationExpr:
		p.CurrentType = Where
		p.WhereOp = s.Op
	case *ast.ColumnName:
		switch p.CurrentType {
		case Where:
			p.WhereColumns = append(p.WhereColumns, s.Name.L)
		default:
			p.Columns = append(p.Columns, s.Name.L)
		}
	case *test_driver.ValueExpr:
		switch p.CurrentType {
		case Where:
			p.WhereValues = append(p.WhereValues, s.GetDatumString())
		default:
			p.Values = append(p.Values, s.GetDatumString())
		}
	}
	return in, false
}

func (p *ParsedQuery) Leave(in ast.Node) (ast.Node, bool) {
	switch in.(type) {
	case *ast.SelectStmt, *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		p.LeaveFunc()
	case *ast.BinaryOperationExpr:
		p.CurrentType = ""
	}
	return in, true
}

func (p *ParsedQuery) LeaveFunc() {
	if f, ok := p.funcs[p.Statement]; ok {
		f(p)
		return
	}
	//log.Printf("%sStmt %s\n", p.Statement, jsonDump(p))
}

/*
func jsonDump(v any) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}
*/

func Parse(sql string) (*ast.StmtNode, error) {
	p := parser.New()

	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, err
	}

	return &stmtNodes[0], nil
}

const (
	User     = "User"
	Password = "Password"
)

var (
	lUser     = strings.ToLower(User)
	lPassword = strings.ToLower(Password)
)

func columnsToConfig(p *ParsedQuery) ([]Server, error) {
	res := []Server{}
	if len(p.Columns) == 0 {
		p.Columns = []string{User, Password}
	}
	for col := 0; col < len(p.Values); col += len(p.Columns) {
		values := p.Values[col:]

		s := Server{}
		for i, column := range p.Columns {
			if len(values) <= i {
				return nil, fmt.Errorf("values length is less than columns length")
			}
			switch strings.ToLower(column) {
			case lUser:
				s.User = values[i]
			case lPassword:
				s.Password = values[i]
			default:
				return nil, fmt.Errorf("column %s not found", column)
			}
		}
		res = append(res, s)
	}
	return res, nil
}

func updateColumns(p *ParsedQuery, s Server) (Server, error) {
	if len(p.Columns) == 0 {
		p.Columns = []string{User, Password}
	}
	values := p.Values
	for i, column := range p.Columns {
		if len(values) <= i {
			return s, fmt.Errorf("values length is less than columns length")
		}
		switch strings.ToLower(column) {
		case lUser:
			s.User = values[i]
		case lPassword:
			s.Password = values[i]
		default:
			return s, fmt.Errorf("column %s not found", column)
		}
	}
	return s, nil
}

func getString(s []string, i int) string {
	return map[bool]func() string{
		true:  func() string { return s[i] },
		false: func() string { return "" },
	}[i < len(s)]()
}

func whereColumnsToConfig(p *ParsedQuery, servers []Server) ([]Server, error) {
	if p.WhereColumns == nil || len(p.WhereColumns) == 0 {
		return servers, nil
	}
	if p.WhereOp != opcode.EQ {
		return nil, errors.New("where only supports equal operation")
	}
	res := []Server{}
	for i, column := range p.WhereColumns {
		switch strings.ToLower(column) {
		case lUser:
			res = selection(func(sv Server) bool { return sv.User == getString(p.WhereValues, i) }, servers)
		case lPassword:
			res = selection(func(sv Server) bool { return sv.Password == getString(p.WhereValues, i) }, servers)
		}
	}
	return res, nil

}

func selection(f func(s Server) bool, servers []Server) []Server {
	res := []Server{}
	for _, s := range servers {
		if f(s) {
			res = append(res, s)
		}
	}
	return res
}

func selectResultset(p *ParsedQuery, servers []Server) ([]string, [][]interface{}, error) {
	if len(p.Columns) == 0 {
		p.Columns = []string{User, Password}
	}
	columns := []string{}
	rows := make([][]interface{}, 0, len(servers))
	for _, s := range servers {
		row := []interface{}{}
		col := []string{}
		for _, column := range p.Columns {
			switch strings.ToLower(column) {
			case lUser:
				row = append(row, s.User)
				col = append(col, User)
			case lPassword:
				row = append(row, s.Password)
				col = append(col, Password)
			}
			columns = col
		}
		rows = append(rows, row)
	}
	return columns, rows, nil
}

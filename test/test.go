package main

import (
	"encoding/json"
	"fmt"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

func parse(sql string) (*ast.StmtNode, error) {
	p := parser.New()

	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, err
	}

	return &stmtNodes[0], nil
}

func main() {
	astNode, err := parse("SELECT username, password FROM users where username = 'user1'")
	//astNode, err := parse("INSERT INTO syain(name1,name2,romaji) VALUES ('bbbbbbb','aaaa','suzuki');")
	//astNode, err := parse("delete from users where name1 = 'bbbbbbb'")
	//astNode, err := parse("UPDATE users SET password = 'bbbbbb' WHERE username = 'user1'")

	if err != nil {
		fmt.Printf("parse error: %v\n", err.Error())
		return
	}
	//log.Printf("%# v\n", *astNode)
	data := Data{
		TableName:    "",
		Columns:      []string{},
		Values:       []string{},
		WhereColumns: []string{},
		WhereValues:  []string{},
	}
	(*astNode).Accept(&data)
}

// 見つけたテーブル名を格納する struct
type Data struct {
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

const (
	UpdateStmt = "udpate"
	InsertStmt = "insert"
	DeleteStmt = "delete"
	SelectStmt = "select"

	Where = "Where"
)

func (v *Data) Enter(in ast.Node) (ast.Node, bool) {
	// Select
	if tab, ok := in.(*ast.SelectStmt); ok {
		v.Statement = SelectStmt
		v.Text = tab.Text()
	}
	// update
	if tab, ok := in.(*ast.UpdateStmt); ok {
		v.Statement = UpdateStmt
		v.Text = tab.Text()
	}

	// insert
	if tab, ok := in.(*ast.InsertStmt); ok {
		v.Statement = InsertStmt
		v.Text = tab.Text()
	}

	if tab, ok := in.(*ast.DeleteStmt); ok {
		v.Statement = DeleteStmt
		v.Text = tab.Text()
	}
	if table, ok := in.(*ast.TableName); ok {
		v.TableName = table.Name.L
	}
	if op, ok := in.(*ast.BinaryOperationExpr); ok {
		//log.Printf("%# v\n", op)
		v.CurrentType = Where
		v.WhereOp = op.Op
	}

	// columnの抽出
	if col, ok := in.(*ast.ColumnName); ok {
		switch v.CurrentType {
		case Where:
			v.WhereColumns = append(v.WhereColumns, col.Name.L)
		default:
			v.Columns = append(v.Columns, col.Name.L)
		}

	}
	// valueの抽出
	if value, ok := in.(*test_driver.ValueExpr); ok {
		switch v.CurrentType {
		case Where:
			v.WhereValues = append(v.WhereValues, value.GetDatumString())
		default:
			v.Values = append(v.Values, value.GetDatumString())
		}

	}
	return in, false
}

func (v *Data) Leave(in ast.Node) (ast.Node, bool) {
	if _, ok := in.(*ast.SelectStmt); ok {
		fmt.Printf("selectStmt %s\n", jsonDump(v))
	}
	if _, ok := in.(*ast.InsertStmt); ok {
		fmt.Printf("insertStmt %s\n", jsonDump(v))
	}
	if _, ok := in.(*ast.DeleteStmt); ok {
		fmt.Printf("deleteStmt %s\n", jsonDump(v))
	}
	if _, ok := in.(*ast.UpdateStmt); ok {
		fmt.Printf("updateStmt %s\n", jsonDump(v))
	}
	if _, ok := in.(*ast.BinaryOperationExpr); ok {
		v.CurrentType = ""
	}
	return in, true
}

func jsonDump(v any) string {
	b, _ := json.MarshalIndent(v, "", " ")
	return string(b)
}

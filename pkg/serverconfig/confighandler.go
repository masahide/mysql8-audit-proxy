package serverconfig

import (
	"fmt"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
)

type configHandler struct {
	server.EmptyHandler
	*Manager
	err error
	res *mysql.Result
}

func (h *configHandler) UseDB(dbName string) error {
	return nil
}

type serverConfig struct {
	Servers map[string]Server
}

func (h *configHandler) selectStmt(p *ParsedQuery) {
	h.res = &mysql.Result{}
	col, data, err := h.Select(p)
	if err != nil {
		h.err = err
		return
	}
	if len(col) == 0 {
		h.res = &mysql.Result{}
		return
	}
	r, _ := mysql.BuildSimpleResultset(col, data, false)
	h.res = &mysql.Result{Resultset: r}
}
func (h *configHandler) insertStmt(p *ParsedQuery) {
	var n uint64
	n, h.err = h.Insert(p)
	h.res = &mysql.Result{AffectedRows: n}

}
func (h *configHandler) updateStmt(p *ParsedQuery) {
	var n uint64
	n, h.err = h.Update(p)
	h.res = &mysql.Result{AffectedRows: n}
}
func (h *configHandler) deleteStmt(p *ParsedQuery) {
	var n uint64
	n, h.err = h.Delete(p)
	h.res = &mysql.Result{AffectedRows: n}
}

func (h *configHandler) handleQuery(query string, binary bool) (*mysql.Result, error) {
	astNode, err := Parse(query)
	if err != nil {
		return nil, err
	}
	data := NewParsedQuery(
		map[string]func(p *ParsedQuery){
			SelectStmt: h.selectStmt,
			InsertStmt: h.insertStmt,
			UpdateStmt: h.updateStmt,
			DeleteStmt: h.deleteStmt,
		},
	)
	(*astNode).Accept(data)
	return h.res, h.err
}

func (h *configHandler) HandleQuery(query string) (*mysql.Result, error) {
	return h.handleQuery(query, false)
}

func (h *configHandler) HandleOtherCommand(cmd byte, data []byte) error {
	return mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d is not supported now", cmd))
}

func NewConfigHandler(m *Manager) *configHandler {
	return &configHandler{Manager: m}
}

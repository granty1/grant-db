package session

import (
	"context"
	"fmt"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"grant-db/kv"
	"grant-db/sessionctx"
	"grant-db/sessionctx/variable"
	"sync"
)

type Session interface {
	sessionctx.Context
	Status() uint16
	Parse(ctx context.Context, sql string) ([]ast.StmtNode, error)
	SetClientCapability(uint32)
	SetConnectionID(connectionID uint64)
}

type session struct {
	store       kv.Storage
	parser      *parser.Parser
	sessionVars *variable.SessionVars
	currentCtx  context.Context

	mu struct {
		sync.RWMutex
		values map[fmt.Stringer]interface{}
	}
}

func NewSession(store kv.Storage) (Session, error) {
	se := &session{
		store:       store,
		parser:      parser.New(),
		sessionVars: variable.NewSessionVars(),
	}
	return se, nil
}

func (s *session) SetConnectionID(connectionID uint64) {
	s.sessionVars.ConnectionID = connectionID
}

func (s *session) SetClientCapability(capability uint32) {
	s.sessionVars.ClientCapability = capability
}

func (s *session) SetValue(key fmt.Stringer, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mu.values[key] = value
}

func (s *session) Value(key fmt.Stringer) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.values[key]
}

func (s *session) GetSessionVars() *variable.SessionVars {
	return s.sessionVars
}

func (s *session) Status() uint16 {
	return s.sessionVars.Status
}

func (s *session) Parse(ctx context.Context, sql string) ([]ast.StmtNode, error) {
	stmts, _, err := s.ParseSQL(ctx, sql, "", "")
	if err != nil {
		return nil, err
	}
	return stmts, nil
}

func (s *session) ParseSQL(ctx context.Context, sql, charset, collation string) ([]ast.StmtNode, []error, error) {
	s.parser.SetSQLMode(0)
	return s.parser.Parse(sql, charset, collation)
}

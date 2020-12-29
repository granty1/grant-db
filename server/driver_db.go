package server

import (
	"context"
	"github.com/pingcap/parser/ast"
	"grant-db/kv"
	"grant-db/session"
)

// GrantDBDriver implements IDriver
type GrantDBDriver struct {
	store kv.Storage
}

// OpenCtx implements IDriver
func (qd *GrantDBDriver) OpenCtx(connID int64, capability uint32, collation uint8, dbname string, tlsState interface{}) (*GrantDBContext, error) {
	s, err := session.NewSession(qd.store)
	if err != nil {
		return nil, err
	}
	s.SetClientCapability(capability)
	s.SetConnectionID(uint64(connID))
	ctx := &GrantDBContext{
		Session:   s,
		currentDB: dbname,
	}
	return ctx, nil
}

// NewGrantDBDriver create a new GrantDBDriver
func NewGrantDBDriver(store kv.Storage) *GrantDBDriver {
	return &GrantDBDriver{
		store: store,
	}
}

// GrantDBContext implements QueryCtx
type GrantDBContext struct {
	session.Session
	currentDB string
	//TODO Grant: GrantDBStatement
}

func (tc *GrantDBContext) ExecuteStmt(ctx context.Context, stmt ast.StmtNode) (ResultSet, error) {
	return nil, nil
}

type GrantDBStatement struct {
	id          uint32
	numParams   int
	boundParams [][]byte
	paramsType  []byte
	ctx         *GrantDBContext
	//TODO Grant: ResultSet
	sql string
}

type grantResultSet struct {
}

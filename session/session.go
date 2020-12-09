package session

import (
	"context"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
)

type Session interface {
	Parse(ctx context.Context, sql string) ([]ast.Node, error)
}

type session struct {
	parser *parser.Parser
}

func (s session) Parse(ctx context.Context, sql string) ([]ast.Node, error) {
	return nil, nil
}
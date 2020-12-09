package sessionctx

import (
	"fmt"
	"grant-db/sessionctx/variable"
)

type Context interface {
	SetValue(key fmt.Stringer, value interface{})
	Value(key fmt.Stringer) interface{}
	GetSessionVars() *variable.SessionVars
}

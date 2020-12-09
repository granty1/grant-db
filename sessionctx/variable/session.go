package variable

import "github.com/pingcap/parser/mysql"

type SessionVars struct {
	// Status stands for the session status
	//. e.g. in transaction or not, auto commit is on or off, and so on.
	Status           uint16
	ClientCapability uint32
	ConnectionID     uint64
}

func NewSessionVars() *SessionVars {
	return &SessionVars{
		Status: mysql.ServerStatusAutocommit,
	}
}

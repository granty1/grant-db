package server

import (
	"grant-db/kv"
)

// GrantDBDriver implements IDriver
type GrantDBDriver struct {
	store kv.Storage
}

// NewGrantDBDriver create a new GrantDBDriver
func NewGrantDBDriver(store kv.Storage) *GrantDBDriver {
	return &GrantDBDriver{
		store: store,
	}
}

// OpenCtx implements IDriver
func (qd *GrantDBDriver) OpenCtx(connID uint64, capability uint32, collation uint8, dbname string, tlsState interface{}) (interface{}, error) {
	return nil, nil
}

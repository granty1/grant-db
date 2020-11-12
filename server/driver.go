package server

// IDriver opens IContext
type IDriver interface {
	// OpenCtx opens an IContext with connection id , client capability, collation ,dbname and optionally the tls state
	OpenCtx(connID uint64, capability uint32, collation uint8, dbname string, tlsState interface{}) (interface{}, error)
}

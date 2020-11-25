package server

import (
	"bufio"
	"net"
)

const  defaultReaderSize = 1024 * 16

type bufferedReadConn struct {
	net.Conn
	rb *bufio.Reader
}

func (conn bufferedReadConn) Read(b []byte) (n int, err error) {
	return conn.rb.Read(b)
}

func newBufferedReadConn(conn net.Conn) *bufferedReadConn {
	return &bufferedReadConn{
		Conn: conn,
		rb:   bufio.NewReaderSize(conn, defaultReaderSize),
	}
}

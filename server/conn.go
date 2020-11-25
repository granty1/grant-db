package server

import (
	"context"
	"grant-db/mysql"
	"grant-db/util/customrand"
	"math/rand"
	"net"
)

type clientConn struct {
	pkt          *packetIO
	bufReadConn  *bufferedReadConn
	conn         net.Conn
	connectionID int64
	server       *Server
	salt         []byte
}

func newClientConn(s *Server, conn net.Conn) *clientConn {
	cc := &clientConn{
		conn:         conn,
		server:       s,
		connectionID: rand.Int63(),
		bufReadConn:  newBufferedReadConn(conn),
	}
	if cc.pkt == nil {
		cc.pkt = newPacketIO(cc.bufReadConn)
	} else {
		cc.pkt.setBufferedReadConn(cc.bufReadConn)
	}
	cc.salt = customrand.Buf(20)
	return cc
}

func (cc *clientConn) handshake(ctx context.Context) error {
	// 1. Initial Handshake
	// Server -> Client
	if err := cc.writeInitialHandshake(ctx); err != nil {
		return err
	}

	// 2. Login Authentication
	// Client -> Server


	// 3. Response Authentication Result
	// Server -> Client
}

func (cc *clientConn) writeInitialHandshake(ctx context.Context) error {
	data := make([]byte, 4, 128)

	// [1] protocol version
	data = append(data, 10)
	// server version string[NUL]
	// [n] end with 0
	data = append(data, mysql.Version...)
	data = append(data, 0)
	// [4] connection id
	data = append(data, byte(cc.connectionID), byte(cc.connectionID>>8), byte(cc.connectionID>>16), byte(cc.connectionID>>24))
	// auth plugin need
	// [8] salt
	data = append(data, cc.salt[:8]...)
	// [1] filler zero bit
	data = append(data, 0)
	// [2] capability flag (lower 2bytes)
	data = append(data, byte(cc.server.capability), byte(cc.server.capability>>8))
	// [1] character set
	// 46 default utf8mb4_bin
	data = append(data, uint8(46))
	// [2] status flags
	AutoCommitStatus := int16(0x0002)
	data = append(data, byte(AutoCommitStatus), byte(AutoCommitStatus>>8))
	// [2] capability flags (upper 2bytes)
	data = append(data, byte(cc.server.capability>>16), byte(cc.server.capability>>24))
	// [1] length of salt
	data = append(data, byte(len(cc.salt)+1))
	// [10] reverse all 00
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	// [n] auth salt part 2
	data = append(data, cc.salt[8:]...)
	data = append(data, 0)
	// [n] auth plugin name
	data = append(data, []byte("mysql_native_password")...)
	data = append(data, 0)
	if err := cc.writePacket(data); err != nil {
		return err
	}
	return cc.flush(ctx)
}

func (cc *clientConn) writePacket(data []byte) error {
	return cc.pkt.writePacket(data)
}

func (cc *clientConn) flush(ctx context.Context) error {
	return cc.pkt.flush()
}

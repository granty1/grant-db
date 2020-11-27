package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"grant-db/mysql"
	"grant-db/util/customrand"
	"io"
	"log"
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
		if err == io.EOF {
			log.Printf("cann't send hankshake due to connection has been ")
		}
		return err
	}

	// 2. Login Authentication
	// Client -> Server
	if err := cc.readOptionalSSLRequestAndHandshakeResponse(ctx); err != nil {
		return err
	}
	// 3. Response Authentication Result
	// Server -> Client
	data := make([]byte, 4, 32)
	// OKHeader => 0x00
	data = append(data, 0x00)
	data = append(data, 0, 0)
	data = append(data, byte(0x0002), byte(0x0002>>8))
	data = append(data, 0, 0)
	err := cc.writePacket(data)
	cc.pkt.sequence = 0
	if err != nil {
		return err
	}
	err = cc.flush(ctx)
	if err != nil {
		return err
	}
	return nil
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

func (cc *clientConn) readOptionalSSLRequestAndHandshakeResponse(ctx context.Context) error {
	data, err := cc.pkt.readPacket()
	if err != nil {
		if err == io.EOF {
			log.Printf("cann't send hankshake due to connection has been ")
		}
		return err
	}

	var resp handshakeResponse41
	var pos int
	if len(data) < 2 {
		log.Println("read response of client in handshake length is too short")
		return errors.New("response of client is too short")
	}

	capability := uint32(binary.LittleEndian.Uint16(data[:2]))
	log.Println(capability & (1 << 9))

	pos, err = parseHandshakeResponseHeader(ctx, &resp, data)
	if err != nil {
		return err
	}

	//TODO Grant: Client SSL Configuration

	// read packet body
	err = parseHandshakeResponseBody(ctx, &resp, data, pos)
	if err != nil {
		return err
	}

	return nil
}

func (cc *clientConn) writePacket(data []byte) error {
	return cc.pkt.writePacket(data)
}

func (cc *clientConn) readPacket() ([]byte, error) {
	return cc.pkt.readPacket()
}

func (cc *clientConn) flush(ctx context.Context) error {
	return cc.pkt.flush()
}

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
	AuthPlugin string
	Attrs      map[string]string
}

func parseHandshakeResponseHeader(ctx context.Context, resp *handshakeResponse41, data []byte) (parseBytes int, err error) {
	//4              capability flags, CLIENT_PROTOCOL_41 always set
	//4              max-packet size
	//1              character set
	//string[23]     reserved (all [0])
	//string[NUL]    username
	if len(data) < 4+4+1+23 {
		return 0, errors.New("response packet format error")
	}
	offset := 0
	capability := binary.LittleEndian.Uint32(data[:4])
	resp.Capability = capability
	offset += 4
	//[4] max packet size
	offset += 4
	//[1] character set
	resp.Collation = data[offset]
	offset++
	//[23] 00
	offset += 23
	return offset, nil
}

func parseHandshakeResponseBody(ctx context.Context, resp *handshakeResponse41, data []byte, offset int) error {
	resp.User = string(data[offset : offset+bytes.IndexByte(data[offset:], 0)])
	offset += len(resp.User) + 1
	log.Println("username:" + resp.User)
	return nil
}

package server

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/pingcap/parser/ast"
	"grant-db/mysql"
	"grant-db/util/customrand"
	"grant-db/util/hack"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

type clientConn struct {
	pkt          *packetIO
	bufReadConn  *bufferedReadConn
	conn         net.Conn
	connectionID int64
	remoteAddr   string
	server       *Server
	salt         []byte
	capability   uint32
	user         string
	dbname       string
	collation    uint8
	attrs        map[string]string
	lastPacket   []byte
	ctx          *GrantDBContext
}

func newClientConn(s *Server, conn net.Conn) *clientConn {
	cc := &clientConn{
		conn:         conn,
		server:       s,
		connectionID: rand.Int63(),
		remoteAddr:   conn.RemoteAddr().String(),
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

func (cc *clientConn) run(ctx context.Context) {
	const size = 4096
	for {
		cc.pkt.setReadTimeout(28800 * time.Second)
		//start := time.Now()
		data, err := cc.readPacket()
		if err != nil {
			log.Printf("[connection:%d] read packet error:%s\n", cc.connectionID, err.Error())
		}

		if err := cc.dispatch(ctx, data); err != nil {
			if err == io.EOF {
				log.Println("client exit")
				return
			}
			log.Println("connection quit, dispatch error:", err.Error())
			break
		}
		cc.pkt.sequence = 0
	}
}

func (cc *clientConn) handleStmt(ctx context.Context, stmt ast.StmtNode, warns interface{}, last bool) error {

	cc.ctx.ExecuteStmt(ctx, stmt)

	return nil
}

func (cc *clientConn) handleQuery(ctx context.Context, sql string) error {
	stmts, err := cc.ctx.Parse(ctx, sql)
	if err != nil {
		log.Println("parse sql error:", err.Error())
		return err
	}

	if len(stmts) == 0 {
		return cc.writeOk(ctx)
	}
	// current only support single-statement
	stmt := stmts[0]
	if err := cc.handleStmt(ctx, stmt, nil, true); err != nil {
		return err
	}

	return cc.writeOk(ctx)
}

func (cc *clientConn) dispatch(ctx context.Context, data []byte) error {
	cc.lastPacket = data
	cmd, data := data[0], data[1:]

	switch cmd {
	case mysql.CmdQuit:
		return io.EOF
	case mysql.CmdQuery:
		if len(data) > 0 && data[len(data)-1] == 0 {
			data = data[:len(data)-1]
		}
		cmdStr := string(hack.String(data))
		log.Println("handle sql: ", cmdStr)
		return cc.handleQuery(ctx, cmdStr)
	}
	return nil
}

func (cc *clientConn) authSwitchRequest(ctx context.Context) ([]byte, error) {
	len := 1 + len("mysql_native_password") + 1 + len(cc.salt) + 1
	data := make([]byte, 4, len)
	data = append(data, 0xfe)
	data = append(data, []byte("mysql_native_password")...)
	data = append(data, byte(0x00))
	data = append(data, cc.salt...)
	data = append(data, 0)
	err := cc.writePacket(data)
	if err != nil {
		return nil, err
	}
	if err := cc.flush(ctx); err != nil {
		return nil, err
	}
	resp, err := cc.readPacket()
	if err != nil {
		return nil, err
	}
	return resp, err
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

	if resp.AuthPlugin == "caching_sha2_password" {
		resp.Auth, err = cc.authSwitchRequest(ctx)
		if err != nil {
			return err
		}
	}

	cc.capability = resp.Capability
	cc.dbname = resp.DBName
	cc.collation = resp.Collation
	cc.attrs = resp.Attrs

	if err := cc.openSessionAndDoAuth(resp.Auth); err != nil {
		return err
	}
	return nil
}

func (cc *clientConn) openSessionAndDoAuth(auth []byte) error {
	//TODO Grant: TLS State Connection
	var err error
	cc.ctx, err = cc.server.driver.OpenCtx(cc.connectionID, cc.capability, cc.collation, cc.dbname, nil)
	if err != nil {
		return nil
	}

	return nil
}

func (cc *clientConn) writePacket(data []byte) error {
	return cc.pkt.writePacket(data)
}

func (cc *clientConn) writeOk(ctx context.Context) error {
	return cc.writeOkWith(ctx, "", 0, 0, cc.ctx.Status(), 0)
}

func (cc *clientConn) writeOkWith(ctx context.Context, msg string, affectedRows, lastInsertID uint64, status, warnCnt uint16) error {
	data := make([]byte, 4, 32)
	data = append(data, mysql.OKHeader)
	data = dumpLengthEncodedInt(data, affectedRows)
	data = dumpLengthEncodedInt(data, lastInsertID)
	data = dumpUint16(data, status)
	data = dumpUint16(data, warnCnt)

	err := cc.writePacket(data)
	if err != nil {
		return err
	}

	return cc.flush(ctx)
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

	if resp.Capability&(1<<21) > 0 {
		num, null, off := parseLengthEncodedInt(data[offset:])
		offset += off
		if !null {
			resp.Auth = data[offset : offset+int(num)]
			offset += int(num)
		}
	} else {
		//TODO Grant: Client Secure Connection
		//TODO Grant: Other Handle
	}
	if resp.Capability&1<<3 > 0 {
		if len(data[offset:]) > 0 {
			idx := bytes.IndexByte(data[offset:], 0)
			resp.DBName = string(data[offset : offset+idx])
			offset += idx + 1
		}
	}
	if resp.Capability&1<<19 > 0 {
		idx := bytes.IndexByte(data[offset:], 0)
		if idx > 0 {
			resp.AuthPlugin = string(data[offset : offset+idx])
		}
		offset += idx + 1
	}
	if resp.Capability&1<<20 > 0 {
		if len(data[offset:]) == 0 {
			return nil
		}
		if num, null, off := parseLengthEncodedInt(data[offset:]); !null {
			offset += off
			rows := data[offset : offset+int(num)]
			attrs, err := parseAttrs(rows)
			if err != nil {
				return err
			}
			resp.Attrs = attrs
		}
	}
	return nil
}

func parseAttrs(data []byte) (map[string]string, error) {
	attrs := make(map[string]string)
	pos := 0
	for pos < len(data) {
		key, _, off, err := parseLengthEncodedBytes(data[pos:])
		if err != nil {
			return attrs, err
		}
		pos += off
		value, _, off, err := parseLengthEncodedBytes(data[pos:])
		if err != nil {
			return attrs, err
		}
		pos += off

		attrs[string(key)] = string(value)
	}
	return attrs, nil
}

// parse unt64 data into []byte
func dumpLengthEncodedInt(buffer []byte, n uint64) []byte {
	switch {
	case n <= 250:
		return append(buffer, byte(n))

	case n <= 0xffff:
		return append(buffer, 0xfc, byte(n), byte(n>>8))

	case n <= 0xffffff:
		return append(buffer, 0xfd, byte(n), byte(n>>8), byte(n>>16))

	case n <= 0xffffffffffffffff:
		return append(buffer, 0xfe, byte(n), byte(n>>8), byte(n>>16), byte(n>>24),
			byte(n>>32), byte(n>>40), byte(n>>48), byte(n>>56))
	}

	return buffer
}

func dumpUint16(buffer []byte, n uint16) []byte {
	buffer = append(buffer, byte(n))
	buffer = append(buffer, byte(n>>8))
	return buffer
}

func dumpUint32(buffer []byte, n uint32) []byte {
	buffer = append(buffer, byte(n))
	buffer = append(buffer, byte(n>>8))
	buffer = append(buffer, byte(n>>16))
	buffer = append(buffer, byte(n>>24))
	return buffer
}

func dumpUint64(buffer []byte, n uint64) []byte {
	buffer = append(buffer, byte(n))
	buffer = append(buffer, byte(n>>8))
	buffer = append(buffer, byte(n>>16))
	buffer = append(buffer, byte(n>>24))
	buffer = append(buffer, byte(n>>32))
	buffer = append(buffer, byte(n>>40))
	buffer = append(buffer, byte(n>>48))
	buffer = append(buffer, byte(n>>56))
	return buffer
}

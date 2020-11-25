package server

import (
	"bufio"
	"errors"
	"fmt"
	"grant-db/mysql"
	"io"
	"time"
)

const defaultWriterSize = 16 * 1024

// packetIO define to read and write data
type packetIO struct {
	bufReadConn *bufferedReadConn
	bufWriter   *bufio.Writer
	sequence    uint8
	readTimeout time.Duration
}

func newPacketIO(bufReadConn *bufferedReadConn) *packetIO {
	return &packetIO{
		sequence:    0,
		bufReadConn: bufReadConn,
		bufWriter:   bufio.NewWriterSize(bufReadConn, defaultWriterSize),
	}
}

func (p *packetIO) setBufferedReadConn(conn *bufferedReadConn) {
	p.bufReadConn = conn
	p.bufWriter = bufio.NewWriterSize(conn, defaultWriterSize)
}

func (p *packetIO) setReadTimeout(timeout time.Duration) {
	p.readTimeout = timeout
}

func (p *packetIO) readOnePacket() ([]byte, error) {
	// Set Read Timeout
	if p.readTimeout > 0 {
		err := p.bufReadConn.SetReadDeadline(time.Now().Add(p.readTimeout))
		if err != nil {
			return nil, err
		}
	}
	// Read Head
	var head [4]byte
	if _, err := io.ReadFull(p.bufReadConn, head[:]); err != nil {
		return nil, err
	}

	// Response Sequence
	sequence := head[3]
	if sequence != p.sequence {
		return nil, errors.New(fmt.Sprintf("invalid sequence, expect:%d, got:%d", p.sequence, sequence))
	}

	// Sequence ++
	p.sequence++

	length := int(uint32(head[0]) | uint32(head[1])<<8 | uint32(head[2])<<16)
	data := make([]byte, length)
	if p.readTimeout > 0 {
		if err := p.bufReadConn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			return nil, err
		}
	}
	if _, err := io.ReadFull(p.bufReadConn, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (p *packetIO) readPacket() ([]byte, error) {
	data, err := p.readOnePacket()
	if err != nil {
		return nil, err
	}

	if len(data) < mysql.MaxPayloadLen {
		//TODO Grant: Metrics Collection

		// Just read one packet
		return data, nil
	}

	// Multi Packet
	for {
		buf, err := p.readOnePacket()
		if err != nil {
			return nil, err
		}
		data = append(data, buf...)
		if len(buf) < mysql.MaxPayloadLen {
			break
		}
	}
	//TODO Grant: Metrics Collection
	return data, nil
}

func (p *packetIO) writePacket(data []byte) error {
	length := len(data) - 4
	////TODO Grant: Metrics Collection

	for length > mysql.MaxPayloadLen {
		// Size is max => 1<<24 -1
		data[0] = 0xff
		data[1] = 0xff
		data[2] = 0xff

		data[3] = p.sequence
		if n, err := p.bufWriter.Write(data[:4+mysql.MaxPayloadLen]); err != nil {
			return err
		} else if n != (4 + mysql.MaxPayloadLen) {
			return errors.New("write packet error")
		} else {
			p.sequence++
			length -= mysql.MaxPayloadLen
			data = data[mysql.MaxPayloadLen:]
		}
	}

	// Write actual size
	data[0] = byte(length)
	data[1] = byte(length >> 8)
	data[2] = byte(length >> 16)
	data[3] = p.sequence

	if n, err := p.bufWriter.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return errors.New("write packet error")
	} else {
		p.sequence++
		return nil
	}
}

func (p *packetIO) flush() error {
	return p.bufWriter.Flush()
}

package server

import (
	"context"
	"grant-db/config"
	"log"
	"net"
)

// Server define the db server
type Server struct {
	listener   net.Listener
	cfg        *config.Config
	capability uint32
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		// default capability
		capability: 1812111,
	}
	var err error
	if s.listener, err = net.Listen("tcp", "127.0.0.1:7878"); err == nil {
		log.Println("server is listening tcp protocol 127.0.0.1:7878")
	} else {
		log.Fatalf("listen port fail: %s\n", err.Error())
	}
	return s
}

func (s *Server) Run() error {

	//TODO Grant: Start Other HTTP Servers

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				log.Println(opErr.Error())
				return nil
			}

			log.Println("accept failed", err.Error())
			return err
		}
		con := s.newConn(conn)
		go s.onConn(con)
	}
}

func (s *Server) onConn(cc *clientConn) {
	log.Printf("[id:%d] connect!", cc.connectionID)
	ctx := context.WithValue(context.Background(), "id", cc.connectionID)
	//TODO Grant: Hand Shake With MySQL Protocol
	if err := cc.handshake(ctx); err != nil {

	}
	//TODO Grant: Conn Run
	for {
		buf := make([]byte, 16)
		n, err := cc.conn.Read(buf)
		if err != nil {
			log.Println("read buf fail:", err.Error())
		}

		log.Printf("read size:%d, content:%s\n", n, string(buf))

	}
}

func (s *Server) newConn(conn net.Conn) *clientConn {
	cc := newClientConn(s, conn)

	//TODO Grant: Set Keep Alive

	//TODO Grant: Set Salt Value
	return cc
}

package server

import (
	"context"
	"grant-db/config"
	"log"
	"net"
	"sync"
)

// Server define the db server
type Server struct {
	*sync.RWMutex

	listener   net.Listener
	cfg        *config.Config
	capability uint32
	driver     IDriver
	clients map[int64]*clientConn
}

func NewServer(cfg *config.Config, driver IDriver) *Server {
	s := &Server{
		cfg: cfg,
		// default capability
		capability: 1812111,
		driver:     driver,
		RWMutex:    &sync.RWMutex{},
		clients:    make(map[int64]*clientConn),
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
	log.Printf("[id:%d][remote addr:%s] connect!", cc.connectionID, cc.remoteAddr)
	ctx := context.WithValue(context.Background(), "id", cc.connectionID)
	//TODO Grant: Hand Shake With MySQL Protocol
	if err := cc.handshake(ctx); err != nil {
		log.Println("handshake error:", err.Error())
		return
	}
	// Record current connected clients
	s.Lock()
	s.clients[cc.connectionID] = cc
	s.Unlock()

	cc.run(ctx)

	log.Printf("[id:%d] connection closed\n", cc.connectionID)
}

func (s *Server) newConn(conn net.Conn) *clientConn {
	cc := newClientConn(s, conn)
	conn.RemoteAddr()
	//TODO Grant: Set Keep Alive

	//TODO Grant: Set Salt Value
	return cc
}

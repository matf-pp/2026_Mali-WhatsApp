package main

import (
	"log"
	"net"
)


type Server struct {
	listenAddr string
	ln         net.Listener
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
	}
}

// Basic Server start
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.ln = ln

	log.Printf("Server started on %s\n", s.listenAddr)

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Printf("Accept error: %s\n", err)
			continue
		}
		
		log.Printf("New connection accepted from %s\n", conn.RemoteAddr())
		// CLose now because there is no client logic
		conn.Close()
	}
}

func main() {
	server := NewServer(":8080")
	log.Fatal(server.Start())
}
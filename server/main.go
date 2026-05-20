package main

import (
	"bufio"
	"log"
	"net"
)

type Client struct {
	ID   string
	Conn net.Conn
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{} 
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln

	log.Printf("Server started on %s\n", s.listenAddr)

	go s.acceptLoop()

	<-s.quitch
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Printf("Accept error: %s\n", err)
			continue
		}

		log.Printf("New connection accepted from %s\n", conn.RemoteAddr())
		
		// Handle each client in a separate goroutine
		go s.handleClient(conn)
	}
}


func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	username, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read username: %s\n", err)
		return
	}

	username = string(username[:len(username)-1])
	if len(username) > 0 && username[len(username)-1] == '\r' {
		username = username[:len(username)-1]
	}

	client := &Client{
		ID:   username,
		Conn: conn,
	}
	_ = client // Just for now, beacause of compiler rules

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Client '%s' disconnected\n", username)
			break
		}
		log.Printf("Received from '%s': %s", username, message)
	}
}

func main() {
	server := NewServer(":8080")
	log.Fatal(server.Start())
}
package main

import (
	"bufio"
	"fmt" 
	"log"
	"net"
	"strings" 
	"sync"
)

type Client struct {
	ID   string
	Conn net.Conn
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	clients    map[string]*Client
	mu         sync.Mutex
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		clients:    make(map[string]*Client),
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
	
	conn.Write([]byte("Username: "))
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read username: %s\n", err)
		return
	}
	username = strings.TrimSpace(username) 
 
	conn.Write([]byte("Password: ")) 
	password, err := reader.ReadString('\n') 
	if err != nil {
		log.Printf("Failed to read password %s\n", err)
		return
	}
	password = strings.TrimSpace(password)

	client := &Client{
		ID:   username,
		Conn: conn,
	}

	s.registerClient(client)
	defer s.deregisterClient(username)

	log.Printf("Client '%s' connected from %s\n", username, conn.RemoteAddr())

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Client '%s' disconnected\n", username)
			break
		}

		s.broadcastMessage(username, message)
	}
}

func (s *Server) registerClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
}

func (s *Server) deregisterClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, id)
	log.Printf("Client '%s' removed from registry\n", id)
}

func (s *Server) broadcastMessage(senderID string, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	formattedMsg := fmt.Sprintf("[%s]: %s", senderID, msg)

	for id, client := range s.clients {
		if id != senderID {
			_, err := client.Conn.Write([]byte(formattedMsg))
			if err != nil {
				log.Printf("Failed to send message to %s: %s\n", id, err)
			}
		}
	}
}

func main() {
	server := NewServer(":8080")
	log.Fatal(server.Start())
}
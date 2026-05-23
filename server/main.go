package main

import (
	"bufio"
	"database/sql"
	"fmt" 
	"log"
	"net"
	"strings" 
	"sync"
	"time"

	_ "modernc.org/sqlite"

)

type Client struct {
	ID   string
	UserID int 
	Conn net.Conn
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	clients    map[string]*Client
	mu         sync.Mutex
	db         *sql.DB 
}

func NewServer(listenAddr string, db *sql.DB) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		clients:    make(map[string]*Client),
		db:			db,
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

	var userID int
    err = s.db.QueryRow(
        "SELECT id FROM users WHERE username = ? AND password = ?",
        username, password,
    ).Scan(&userID)
    if err != nil {
        if err == sql.ErrNoRows {
            log.Printf("Permission denied for '%s': wrong username or password.\n", username)
            conn.Write([]byte("Error: wrong username or password.\n"))
            return
        }
        log.Printf("DB error during registration: %s\n", err)
        return
    }

	client := &Client{
		ID:   username,
		UserID: userID,
		Conn: conn,
	}

	s.registerClient(client)
	defer func() {
        s.deregisterClient(username)
        log.Printf("User '%s' logged out.\n", username)
        s.broadcastSystem(fmt.Sprintf("*** %s has left the chat ***\n", username))
    }()

	log.Printf("User '%s' (ID: %d) has connected.\n", username, userID)
    conn.Write([]byte(fmt.Sprintf("Welcome, %s! Format: @receiver msg | broadcast: msg\n", username)))
    s.broadcastSystem(fmt.Sprintf("*** %s has joined the chat ***\n", username))

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		s.saveBroadcastMessage(client, message)

		s.broadcastMessage(username, message)
	}
}

func (s *Server) saveBroadcastMessage(sender *Client, msg string) {
	_, err := s.db.Exec(
		"INSERT INTO chat (idSender, idReceiver, messageText, time) VALUES (?, NULL, ?, ?)",
		sender.UserID, msg, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		log.Printf("Error saving broadcast message: %s\n", err)
	}
}

func (s *Server) broadcastSystem(msg string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    for _, client := range s.clients {
        client.Conn.Write([]byte(msg))
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
}

func (s *Server) broadcastMessage(senderID string, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	formattedMsg := fmt.Sprintf("[%s]: %s\n", senderID, msg)

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
	db, err := sql.Open("sqlite", "./chat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
 
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal(err)
	}
	
	server := NewServer(":8080", db)
	log.Fatal(server.Start())
}
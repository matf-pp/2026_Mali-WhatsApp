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
	PublicKey string  // new
}

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	clients    map[string]*Client
	mu         sync.Mutex
	db         *sql.DB 
}

type ChatMessage struct {
    Text     string
    Received bool   // true - for sender, false for receiver
    Time     string
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


	message, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	message = strings.TrimSpace(message)

	if strings.HasPrefix(message, "HANDSHAKE:") {
		client.PublicKey = strings.TrimPrefix(message, "HANDSHAKE:")
		
		s.mu.Lock()
		for id, c := range s.clients {
			if id != username {
				// forward our key to the other client
				c.Conn.Write([]byte(fmt.Sprintf("NJEGOV_KLJUC:%s\n", client.PublicKey)))
				// if the other client already sent their key, send it back to us
				if c.PublicKey != "" {
					conn.Write([]byte(fmt.Sprintf("NJEGOV_KLJUC:%s\n", c.PublicKey)))
				}
			}
		}
		s.mu.Unlock()
	}


	history, err := s.loadHistory(client.UserID)
	if err != nil {
		log.Printf("Failed to load history for %s: %s\n", username, err)
	} else {
		conn.Write([]byte("--- Chat History ---\n"))
		for _, msg := range history {
			direction := "→"
			if msg.Received {
				direction = "←"
			}
			conn.Write([]byte(fmt.Sprintf("[%s] %s %s\n", msg.Time, direction, msg.Text)))
		}
		conn.Write([]byte("--- End of History ---\n"))
	}


	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		if strings.HasPrefix(message, "@") {
			parts := strings.SplitN(message[1:], " ", 2)
			if len(parts) < 2 || parts[1] == "" {
				conn.Write([]byte("Format: @user msg\n"))
				continue
			}
			if err := s.sendDirectMessage(client, parts[0], parts[1]); err != nil {
				conn.Write([]byte(fmt.Sprintf("Error: %s\n", err)))
			}
		} else {
			s.saveBroadcastMessage(client, message)
			s.broadcastMessage(username, message)
		}
	}
}

func (s *Server) sendDirectMessage(sender *Client, recipientName string, text string) error {
	var recipientID int
	err := s.db.QueryRow("SELECT id FROM users WHERE username = ?", recipientName).Scan(&recipientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("User '%s' does not exist", recipientName)
		}
		return fmt.Errorf("DB error: %s", err)
	}
 
	_, err = s.db.Exec(
		"INSERT INTO chat (idSender, idReceiver, messageText, time) VALUES (?, ?, ?, ?)",
		sender.UserID, recipientID, text, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return fmt.Errorf("Error during message saving: %s", err)
	}
 
	s.mu.Lock()
	recipient, online := s.clients[recipientName]
	s.mu.Unlock()
 
	if online {
		recipient.Conn.Write([]byte(fmt.Sprintf("[DM from %s]: %s\n", sender.ID, text)))
	}
 
	status := "delivered"
	if !online {
		status = "sent (user offline)"
	}
	sender.Conn.Write([]byte(fmt.Sprintf("[You -> %s] (%s): %s\n", recipientName, status, text)))
 
	log.Printf("DM: %s -> %s: %s\n", sender.ID, recipientName, text)
	return nil
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

func (s *Server) loadHistory(userID int) ([]ChatMessage, error) {
    rows, err := s.db.Query(`
        SELECT messageText, (idSender != ?), time 
        FROM chat 
        WHERE idReceiver IS NOT NULL AND (idReceiver = ? OR idSender = ?)
        ORDER BY time ASC`,
        userID, userID, userID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var history []ChatMessage
    for rows.Next() {
        var msg ChatMessage
        if err := rows.Scan(&msg.Text, &msg.Received, &msg.Time); err != nil {
            return nil, err
        }
        history = append(history, msg)
    }
    return history, nil
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
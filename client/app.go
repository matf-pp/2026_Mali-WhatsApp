package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	conn   net.Conn
	reader *bufio.Reader
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Login(username, password, serverAddr string) (string, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return "", fmt.Errorf("Server nedostupan: %w", err)
	}
	a.conn = conn
	a.reader = bufio.NewReader(conn)

	// Čitanje i slanje kredencijala (prilagođeno trenutnom serveru tvojih kolega)
	_, _ = a.reader.ReadString(':')
	_, _ = fmt.Fprintf(a.conn, "%s\n", username)

	_, _ = a.reader.ReadString(':')
	_, _ = fmt.Fprintf(a.conn, "%s\n", password)

	odgovor, err := a.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("Greška pri čitanju odgovora: %w", err)
	}

	if strings.Contains(odgovor, "Error") {
		a.conn.Close()
		return "", fmt.Errorf(strings.TrimSpace(odgovor))
	}

	_, _ = a.reader.ReadString('\n') // Čisti welcome poruku

	go a.listenForMessages()
	return "Uspešno logovanje!", nil
}

func (a *App) SendMessage(text string) error {
	if a.conn == nil {
		return fmt.Errorf("Niste povezani")
	}
	_, err := fmt.Fprintf(a.conn, "%s\n", text)
	return err
}

func (a *App) listenForMessages() {
	for {
		poruka, err := a.reader.ReadString('\n')
		if err != nil {
			runtime.EventsEmit(a.ctx, "server_status", "Veza prekinuta.")
			break
		}
		poruka = strings.TrimSpace(poruka)
		if poruka != "" {
			runtime.EventsEmit(a.ctx, "nova_poruka", poruka)
		}
	}
}

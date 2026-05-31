package main

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"net"
	"strings"

	"mini-whatsapp/crypto"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
    ctx               context.Context
    conn              net.Conn
    reader            *bufio.Reader
    sessionKeys       map[string][]byte
    mojPrivatniKljuc *big.Int
}

func NewApp() *App {
    return &App{
        sessionKeys: make(map[string][]byte),
    }
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

	// Čitanje i slanje kredencijala
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

	fmt.Println("Započinjem Diffie-Hellman Handshake...")

	mojPrivatniKljuc, err := crypto.LoadPrivateKey(username)

	if err != nil {
		fmt.Println("Privatni ključ ne postoji, generišem novi...")

		mojPrivatniKljuc, err = crypto.GeneratePrivateKey()
		if err != nil {
			return "", fmt.Errorf("greška pri generisanju DH privatnog ključa: %w", err)
		}

		err = crypto.SavePrivateKey(username, mojPrivatniKljuc)
		if err != nil {
			return "", fmt.Errorf("greška pri čuvanju DH privatnog ključa: %w", err)
		}
	} else {
		fmt.Println("Učitan postojeći DH privatni ključ.")
	}

	a.mojPrivatniKljuc = mojPrivatniKljuc

	mojJavniKljuc := crypto.GeneratePublicKey(mojPrivatniKljuc)

	handshakePoruka := crypto.NewHandshakeMessage(mojJavniKljuc)
	_, err = fmt.Fprintf(a.conn, "HANDSHAKE:%s\n", handshakePoruka.PublicKey)
	if err != nil {
		return "", fmt.Errorf("greška pri slanju handshake poruke: %w", err)
	}
	// --- KRAJ DIFFIE-HELLMAN HANDSHAKE ---
	// ne blokiramo vise, NJEGOV_KLJUC ce stici u listenForMessages

	go a.listenForMessages()
	return "Uspešno logovanje!", nil
}

func (a *App) SendMessage(text string) error {
    if a.conn == nil {
        return fmt.Errorf("niste povezani na server")
    }

    if !strings.HasPrefix(text, "@") {
        return fmt.Errorf("broadcast nije podržan, koristite format: @korisnik poruka")
    }

    parts := strings.SplitN(text, " ", 2)
    if len(parts) != 2 || parts[1] == "" {
        return fmt.Errorf("format: @korisnik poruka")
    }

    primalac := strings.TrimPrefix(parts[0], "@")
    cistTekst := parts[1]

    key, ok := a.sessionKeys[primalac]
    if !ok {
        return fmt.Errorf("nemate AES session key za korisnika %s", primalac)
    }

    enkriptovanaStruktura, err := crypto.NewEncryptedMessage(key, cistTekst)
    if err != nil {
        return fmt.Errorf("greška pri kriptovanju DM-a: %w", err)
    }

    porukaZaMrezu := fmt.Sprintf("@%s %s:%s", primalac, enkriptovanaStruktura.Nonce, enkriptovanaStruktura.Ciphertext)

    _, err = fmt.Fprintf(a.conn, "%s\n", porukaZaMrezu)
    return err
}

func (a *App) listenForMessages() {
     for {
         poruka, err := a.reader.ReadString('\n')
         if err != nil {
             break
         }

         poruka = strings.TrimSpace(poruka)

         if strings.HasPrefix(poruka, "NJEGOV_KLJUC:") {
             data := strings.TrimPrefix(poruka, "NJEGOV_KLJUC:")
             parts := strings.SplitN(data, ":", 2)

             if len(parts) == 2 {
                 username := parts[0]
                 kljucString := parts[1]

                 tudjiKljuc, err := crypto.PublicKeyFromString(kljucString)
                 if err == nil {
                     a.sessionKeys[username] = crypto.CreateSessionKey(a.mojPrivatniKljuc, tudjiKljuc)
                     fmt.Println("AES ključ kreiran za korisnika:", username)
					 _, _ = fmt.Fprintf(a.conn, "GET_HISTORY\n")
                 }
             }

             continue
         }

         if strings.HasPrefix(poruka, "[HISTORY from ") || strings.HasPrefix(poruka, "[HISTORY to ") {
             delovi := strings.SplitN(poruka, "]: ", 2)
             if len(delovi) != 2 {
                 runtime.EventsEmit(a.ctx, "nova_poruka", poruka)
                 continue
             }

             prefiks := delovi[0]
             kriptoTekst := delovi[1]

             var username string
             if strings.HasPrefix(prefiks, "[HISTORY from ") {
                 username = strings.TrimPrefix(prefiks, "[HISTORY from ")
             } else {
                 username = strings.TrimPrefix(prefiks, "[HISTORY to ")
             }

             key, ok := a.sessionKeys[username]
             if !ok {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Nema AES ključa za dešifrovanje istorije]", prefiks))
                 continue
             }

             kriptoDelovi := strings.SplitN(kriptoTekst, ":", 2)
             if len(kriptoDelovi) != 2 {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Neispravan format šifrovane poruke]", prefiks))
                 continue
             }

             msgStruktura := crypto.EncryptedMessage{
                 Nonce:      kriptoDelovi[0],
                 Ciphertext: kriptoDelovi[1],
             }

             cistTekst, err := crypto.OpenEncryptedMessage(key, msgStruktura)
             if err != nil {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Greška pri dešifrovanju istorije]", prefiks))
                 continue
             }

             runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: %s", prefiks, cistTekst))
             continue
         }

         if strings.HasPrefix(poruka, "[DM from ") {
             delovi := strings.SplitN(poruka, "]: ", 2)
             if len(delovi) != 2 {
                 runtime.EventsEmit(a.ctx, "nova_poruka", poruka)
                 continue
             }

             prefiks := delovi[0]
             kriptoTekst := delovi[1]

             username := strings.TrimPrefix(prefiks, "[DM from ")

             key, ok := a.sessionKeys[username]
             if !ok {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Nema AES ključa za dešifrovanje]", prefiks))
                 continue
             }

             kriptoDelovi := strings.SplitN(kriptoTekst, ":", 2)
             if len(kriptoDelovi) != 2 {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Neispravan format šifrovane poruke]", prefiks))
                 continue
             }

             msgStruktura := crypto.EncryptedMessage{
                 Nonce:      kriptoDelovi[0],
                 Ciphertext: kriptoDelovi[1],
             }

             cistTekst, err := crypto.OpenEncryptedMessage(key, msgStruktura)
             if err != nil {
                 runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: [Greška pri dešifrovanju]", prefiks))
                 continue
             }

             runtime.EventsEmit(a.ctx, "nova_poruka", fmt.Sprintf("%s]: %s", prefiks, cistTekst))
             continue
         }

         if strings.HasPrefix(poruka, "[You -> ") {
            continue
         }

         runtime.EventsEmit(a.ctx, "nova_poruka", poruka)
     }
}
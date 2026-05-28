package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

	"mini-whatsapp/crypto"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	conn     net.Conn
	reader   *bufio.Reader
	aesKljuc []byte
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

	mojPrivatniKljuc, err := crypto.GeneratePrivateKey()
	if err != nil {
		return "", fmt.Errorf("greška pri generisanju DH privatnog ključa: %w", err)
	}

	mojJavniKljuc := crypto.GeneratePublicKey(mojPrivatniKljuc)

	handshakePoruka := crypto.NewHandshakeMessage(mojJavniKljuc)
	_, err = fmt.Fprintf(a.conn, "HANDSHAKE:%s\n", handshakePoruka.PublicKey)
	if err != nil {
		return "", fmt.Errorf("greška pri slanju handshake poruke: %w", err)
	}
	// 4. Čitamo sa mreže javni ključ od drugog korisnika koji nam šalje server
	// Pretpostavljamo da server šalje liniju u formatu: "NJEGOV_KLJUC:vrednost"
	odgovorHandshake, err := a.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("greška pri primanju tuđeg javnog ključa: %w", err)
	}
	odgovorHandshake = strings.TrimSpace(odgovorHandshake)

	tudjiJavniKljucString := strings.Replace(odgovorHandshake, "NJEGOV_KLJUC:", "", 1)

	tudjiJavniKljuc, err := crypto.PublicKeyFromString(tudjiJavniKljucString)
	if err != nil {
		return "", fmt.Errorf("nevalidan javni ključ sa servera: %w", err)
	}

	// u pozadini radi: KDF(DH_tajna) i vraća 32 bajta za AES
	a.aesKljuc = crypto.CreateSessionKey(mojPrivatniKljuc, tudjiJavniKljuc)
	fmt.Println("Diffie-Hellman uspešan! AES ključ sesije je bezbedno kreiran.")
	// --- KRAJ DIFFIE-HELLMAN HANDSHAKE ---

	go a.listenForMessages()
	return "Uspešno logovanje!", nil
}

func (a *App) SendMessage(text string) error {
	if a.conn == nil {
		return fmt.Errorf("niste povezani na server")
	}

	var porukaZaMrezu string

	if strings.HasPrefix(text, "@") {
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			primalac := parts[0] // npr. "@Bob"
			cistTekst := parts[1]

			enkriptovanaStruktura, err := crypto.NewEncryptedMessage(a.aesKljuc, cistTekst)
			if err != nil {
				return fmt.Errorf("greška pri kriptovanju DM-a: %s", err)
			}

			porukaZaMrezu = fmt.Sprintf("%s %s:%s", primalac, enkriptovanaStruktura.Nonce, enkriptovanaStruktura.Ciphertext)
		} else {
			porukaZaMrezu = text
		}
	} else {
		enkriptovanaStruktura, err := crypto.NewEncryptedMessage(a.aesKljuc, text)
		if err != nil {
			return fmt.Errorf("greška pri kriptovanju javne poruke: %s", err)
		}
		porukaZaMrezu = fmt.Sprintf("%s:%s", enkriptovanaStruktura.Nonce, enkriptovanaStruktura.Ciphertext)
	}

	_, err := fmt.Fprintf(a.conn, "%s\n", porukaZaMrezu)
	return err
}

func (a *App) listenForMessages() {
	for {
		//'a.' jer je polje strukture
		poruka, err := a.reader.ReadString('\n')
		if err != nil {
			break
		}
		poruka = strings.TrimSpace(poruka)

		var prikaznaPoruka string

		if strings.Contains(poruka, "[DM from") && strings.Contains(poruka, ":") {
			delovi := strings.SplitN(poruka, "]: ", 2)
			if len(delovi) == 2 {
				prefiks := delovi[0]
				kriptoDelovi := strings.SplitN(delovi[1], ":", 2)

				if len(kriptoDelovi) == 2 {
					msgStruktura := crypto.EncryptedMessage{
						Nonce:      kriptoDelovi[0],
						Ciphertext: kriptoDelovi[1],
					}
					cistTekst, err := crypto.OpenEncryptedMessage(a.aesKljuc, msgStruktura)
					if err == nil {
						prikaznaPoruka = fmt.Sprintf("%s]: %s", prefiks, cistTekst)
					} else {
						prikaznaPoruka = fmt.Sprintf("%s]: [Greška pri dešifrovanju]", prefiks)
					}
				}
			}
		} else if strings.Contains(poruka, "[") && strings.Contains(poruka, "]:") {
			delovi := strings.SplitN(poruka, "]: ", 2)
			if len(delovi) == 2 {
				prefiks := delovi[0]
				kriptoDelovi := strings.SplitN(delovi[1], ":", 2)

				if len(kriptoDelovi) == 2 {
					msgStruktura := crypto.EncryptedMessage{
						Nonce:      kriptoDelovi[0],
						Ciphertext: kriptoDelovi[1],
					}
					cistTekst, err := crypto.OpenEncryptedMessage(a.aesKljuc, msgStruktura)
					if err == nil {
						prikaznaPoruka = fmt.Sprintf("%s]: %s", prefiks, cistTekst)
					} else {
						prikaznaPoruka = fmt.Sprintf("%s]: [Greška pri dešifrovanju]", prefiks)
					}
				}
			}
		} else {
			prikaznaPoruka = poruka
		}

		runtime.EventsEmit(a.ctx, "nova_poruka", prikaznaPoruka)
	}
}

# Mini WhatsApp

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/50d8613871eb46f6a8f67ee791c8b8ba)](https://app.codacy.com/gh/matf-pp/2026_Mali-WhatsApp/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

## Opis
Implementacija aplikacije za razmenu poruka između korisnika, uz korišćenje 
enkripcije i dekripcije poruka. Aplikacija koristi Diffie-Hellman razmenu 
ključeva i AES-GCM enkripciju za bezbednu komunikaciju između korisnika.

## Jezici i tehnologije
- **Go** — server i klijentska logika
- **Wails** — desktop GUI framework
- **SQLite3** — baza podataka za čuvanje poruka i korisnika

## Potrebno za pokretanje
- Go 1.21+
- Wails (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- SQLite3

## Prevođenje i pokretanje

### Razvojno okruženje
```bash
# Server
cd server
go run main.go

# Klijent
cd client
wails dev
```

### Izvršni fajlovi
```bash
# Server
cd server
go build -o server main.go
./server

# Klijent
cd client
wails build
./build/bin/mini-whatsapp
```

### Inicijalizacija baze
```bash
    #Iz korenog direktorijuma pokrenuti:
    make
```

## Operativni sistem
Izvršni fajlovi su napravljeni za **Linux (x86_64)**, testirano na Ubuntu 24.04.

## Autori
| Ime | Email |
|-----|-------|
| Nikolaj Molčanov | mi23239@alas.matf.bg.ac.rs |
| Stefan Gajić | mi23056@alas.matf.bg.ac.rs |
| Dušan Marić | mi23065@alas.matf.bg.ac.rs |

## Napomene
- Server se pokreće pre klijenta
- Istorija poruka će biti vidljiva tek nakon što oba korisnika urade login
- Podrazumevani korisnici: `Alice`/`tajna456` i `Bob`/`sifra123`
- Server sluša na portu `8080`

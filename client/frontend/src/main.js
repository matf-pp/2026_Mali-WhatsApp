import { Login, SendMessage } from '../wailsjs/go/main/App.js';
import { EventsOn } from '../wailsjs/runtime/runtime.js';

let trenutniKorisnik = "";

// Slušamo poruke sa Go backenda
EventsOn("nova_poruka", (poruka) => {
    prikažiPorukuUEkranu(poruka);
});

EventsOn("server_status", (status) => {
    alert("Status mreže: " + status);
});

// Funkcija za Login
window.izvrsiPrijavu = async function() {
    const user = document.getElementById("username").value.trim();
    const pass = document.getElementById("password").value.trim();
    const addr = document.getElementById("server-addr").value.trim();

    if (!user || !pass || !addr) {
        alert("Popunite sva polja!");
        return;
    }

    try {
        await Login(user, pass, addr);
        trenutniKorisnik = user;
        
        document.getElementById("chat-title").innerText = `Korisnik: ${trenutniKorisnik}`;
        document.getElementById("login-container").style.display = "none";
        document.getElementById("chat-container").style.display = "block";
    } catch (err) {
        alert("Greška: " + err);
    }
}

// Funkcija za slanje poruke
window.posaljiPoruku = async function() {
    const input = document.getElementById("message-input");
    const tekst = input.value.trim();

    if (tekst === "") return;

    try {
        await SendMessage(tekst);
        
        // Lokalni prikaz poslate poruke
        if (tekst.startsWith("@")) {
            prikažiPorukuUEkranu(`[Ti -> ${tekst.split(' ')[0].substring(1)}]: ${tekst.substring(tekst.indexOf(' ') + 1)}`, true);
        } else {
            prikažiPorukuUEkranu(`[Ti]: ${tekst}`, true);
        }
        
        input.value = "";
    } catch (err) {
        alert("Greška pri slanju: " + err);
    }
}

// Pomoćna funkcija za ispis na ekranu
function prikažiPorukuUEkranu(tekst, daLiSamJaPoslao = false) {
    const prozorSaPorukama = document.getElementById("chat-messages"); // ISPRAVLJENO OVDE
    const divPoruka = document.createElement("div");
    
    // Dodajemo osnovnu klasu za poruku
    divPoruka.className = "msg";
    divPoruka.innerText = tekst;
    
    // Ako je poruka naša (ili počinje sa [Ti]), dodajemo klasu koja je boji u plavo i gura desno
    if (daLiSamJaPoslao || tekst.startsWith("[Ti]")) {
        divPoruka.classList.add("moja-poruka");
    }
    
    prozorSaPorukama.appendChild(divPoruka); // ISPRAVLJENO OVDE
    prozorSaPorukama.scrollTop = prozorSaPorukama.scrollHeight; // ISPRAVLJENO OVDE
}

// Slanje na Enter taster
document.getElementById("message-input")?.addEventListener("keypress", function(e) {
    if (e.key === "Enter") {
        window.posaljiPoruku();
    }
});
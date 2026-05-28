import { Login, SendMessage } from '../wailsjs/go/main/App.js';
import { EventsOn } from '../wailsjs/runtime/runtime.js';

let trenutniKorisnik = "";

EventsOn("nova_poruka", (poruka) => {
    prikažiPorukuUEkranu(poruka);
});

EventsOn("server_status", (status) => {
    alert("Status: " + status);
});

//Login dugme
window.izvrsiPrijavu = async function() {
    const user = document.getElementById("username").value.trim();
    const pass = document.getElementById("password").value.trim();
    const addr = document.getElementById("server-addr").value.trim();

    if (!user || !pass || !addr) {
        alert("Molimo unesite sva polja!");
        return;
    }

    try {
        // Pozivamo Go metodu Login iz app.go
        const rezultat = await Login(user, pass, addr);
        trenutniKorisnik = user;
        
        // Promeni naslov četa da piše ko je ulogovan
        document.getElementById("chat-title").innerText = `Ulogovan kao: ${trenutniKorisnik}`;

        // Sakrij login formu, prikaži čet prozor
        document.getElementById("login-container").style.display = "none";
        document.getElementById("chat-container").style.display = "flex";
    } catch (err) {
        alert("Greška pri prijavi: " + err);
    }
}

window.posaljiPoruku = async function() {
    const input = document.getElementById("message-input");
    const tekst = input.value.trim();

    if (tekst === "") return;

    try {
        // Pozivamo Go metodu SendMessage iz app.go
        await SendMessage(tekst);
        
        // Ako je poruka bila namenjena nekom drugom (@Korisnik), prikazaćemo je odmah i kod nas lokalno
        if (tekst.startsWith("@")) {
            prikažiPorukuUEkranu(`[Ti -> ${tekst.split(' ')[0].substring(1)}]: ${tekst.substring(tekst.indexOf(' ') + 1)}`);
        } else {
            prikažiPorukuUEkranu(`[Ti]: ${tekst}`);
        }
        
        input.value = ""; // Praznimo polje nakon uspešnog slanja
    } catch (err) {
        alert("Greška pri slanju: " + err);
    }
}

// Pomoćna funkcija za ubacivanje oblačića u čet prozor
function prikažiPorukuUEkranu(tekst) {
    const mrezaPoruka = document.getElementById("chat-messages");
    
    const oblačić = document.createElement("div");
    oblačić.className = "msg-bubble";
    oblačić.innerText = tekst;
    
    // Ako poruka počinje sa "[Ti", pomeri oblačić desno i promeni mu boju u svetlozelenu (kao na WhatsApp-u)
    if (tekst.startsWith("[Ti]")) {
        oblačić.style.alignSelf = "flex-end";
        oblačić.style.backgroundColor = "#d9fdd3";
    } else if (tekst.startsWith("[Ti ->")) {
        oblačić.style.alignSelf = "flex-end";
        oblačić.style.backgroundColor = "#e8cbf5"; // Drugačija boja za DM koji ti šalješ
    }
    
    mrezaPoruka.appendChild(oblačić);
    
    // Automatski skroluj na dno prozora kada stigne nova poruka
    mrezaPoruka.scrollTop = mrezaPoruka.scrollHeight;
}

// Omogućavamo da se poruka pošalje i pritiskom na taster Enter na tastaturi
document.getElementById("message-input")?.addEventListener("keypress", function(event) {
    if (event.key === "Enter") {
        event.preventDefault();
        window.posaljiPoruku();
    }
});

window.toggleSifru = function() {
    const passwordInput = document.getElementById("password");
    const toggleButton = document.getElementById("toggle-password");

    if (passwordInput.type === "password") {
        passwordInput.type = "text";
        toggleButton.innerText = "🔒";
    } else {
        passwordInput.type = "password";
        toggleButton.innerText = "👁️";
    }
}
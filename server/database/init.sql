DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS chat;

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chat (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    idSender INTEGER,
    idReceiver INTEGER,
    messageText TEXT NOT NULL,
    time DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(idSender) REFERENCES users(id),
    FOREIGN KEY(idReceiver) REFERENCES users(id)
);

INSERT INTO users (username, password) VALUES ('Bob', 'sifra123');
INSERT INTO users (username, password) VALUES ('Alice', 'tajna456');


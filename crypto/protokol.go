package crypto

import (
	"encoding/base64"
	"errors"
	"math/big"
)

type HandshakeMessage struct {
	Type      string `json:"type"`
	PublicKey string `json:"publicKey"`
}

type EncryptedMessage struct {
	Type       string `json:"type"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func PublicKeyToString(publicKey *big.Int) string {
	return publicKey.String()
}

func PublicKeyFromString(publicKeyString string) (*big.Int, error) {
	publicKey, ok := new(big.Int).SetString(publicKeyString, 10)
	if !ok {
		return nil, errors.New("invalid public key")
	}

	if publicKey.Sign() <= 0 || publicKey.Cmp(P) >= 0 {
		return nil, errors.New("public key out of range")
	}

	return publicKey, nil
}

func CreateSessionKey(myPrivateKey *big.Int, otherPublicKey *big.Int) []byte {
	sharedSecret := ComputeSharedSecret(myPrivateKey, otherPublicKey)
	return DeriveAESKey(sharedSecret)
}

func NewHandshakeMessage(publicKey *big.Int) HandshakeMessage {
	return HandshakeMessage{
		Type:      "handshake",
		PublicKey: PublicKeyToString(publicKey),
	}
}

func NewEncryptedMessage(key []byte, plaintext string) (*EncryptedMessage, error) {
	nonce, ciphertext, err := EncryptMessage(key, []byte(plaintext))
	if err != nil {
		return nil, err
	}

	return &EncryptedMessage{
		Type:       "message",
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func OpenEncryptedMessage(key []byte, msg EncryptedMessage) (string, error) {
	nonce, err := base64.StdEncoding.DecodeString(msg.Nonce)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(msg.Ciphertext)
	if err != nil {
		return "", err
	}

	plaintext, err := DecryptMessage(key, nonce, ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)


func EncryptMessage(key []byte, plaintext []byte) ([]byte, []byte, error) {
	
	block, err := aes.NewCipher(key) 
	if err != nil {
		return nil, nil, err
	}

	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize()) 

	_, err = io.ReadFull(rand.Reader, nonce) 
	if err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil) 

	return nonce, ciphertext, nil
}

func DecryptMessage(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil) 
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
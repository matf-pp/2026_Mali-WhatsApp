package crypto

import (
	"crypto/sha256"
	"math/big"
)

func DeriveAESKey(sharedSecret *big.Int) []byte {
	hash := sha256.Sum256(sharedSecret.Bytes())

	return hash[:]
}
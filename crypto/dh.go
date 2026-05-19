package crypto

import (
	"crypto/rand"
	"errors"
	"math/big"
)

//2048 bit prime
var P, _ = new(big.Int).SetString(
	"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
		"670C354E4ABC9804F1746C08CA237327FFFFFFFFFFFFFFFF",
	16,
)

var G = big.NewInt(2)

func GeneratePrivateKey() (*big.Int, error) {
	if P == nil {
		return nil, errors.New("prime P is not set")
	}

	
	max := new(big.Int).Sub(P, big.NewInt(2))

	privateKey, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}

	
	privateKey.Add(privateKey, big.NewInt(1))

	return privateKey, nil
}


func GeneratePublicKey(privateKey *big.Int) *big.Int {
	publicKey := new(big.Int).Exp(G, privateKey, P)
	return publicKey
}


func ComputeSharedSecret(privateKey *big.Int, otherPublicKey *big.Int) *big.Int {
	sharedSecret := new(big.Int).Exp(otherPublicKey, privateKey, P)
	return sharedSecret
}



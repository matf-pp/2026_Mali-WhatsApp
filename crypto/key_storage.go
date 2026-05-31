package crypto

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

func SavePrivateKey(username string, key *big.Int) error {
	err := os.MkdirAll("keys", 0700)
	if err != nil {
		return err
	}

	filePath := filepath.Join("keys", fmt.Sprintf("%s.key", username))

	return os.WriteFile(filePath, []byte(key.String()), 0600)
}

func LoadPrivateKey(username string) (*big.Int, error) {
	filePath := filepath.Join("keys", fmt.Sprintf("%s.key", username))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	keyString := strings.TrimSpace(string(data))

	key, ok := new(big.Int).SetString(keyString, 10)
	if !ok {
		return nil, fmt.Errorf("invalid private key format")
	}

	return key, nil
}
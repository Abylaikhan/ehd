package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// randomToken — 32 случайных байта в hex (opaque session token).
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken — sha256(token) в hex; в БД хранится хеш, а не сам токен.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

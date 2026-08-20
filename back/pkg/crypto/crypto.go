// Package crypto — шифрование персональных полей (AES-256-GCM) и HMAC для ИИН.
// Ключи приходят из конфигурации (env), вне БД (требование ТЗ, раздел «Безопасность»).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

type Cipher struct {
	aead    cipher.AEAD
	hmacKey []byte
}

// New создаёт Cipher из hex-ключа AES-256 (64 hex-символа = 32 байта) и ключа HMAC.
func New(encKeyHex, hmacKey string) (*Cipher, error) {
	key, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return nil, errors.New("crypto: AUTH_ENC_KEY must be hex")
	}
	if len(key) != 32 {
		return nil, errors.New("crypto: AUTH_ENC_KEY must decode to 32 bytes (AES-256)")
	}
	if hmacKey == "" {
		return nil, errors.New("crypto: AUTH_HMAC_KEY is empty")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead, hmacKey: []byte(hmacKey)}, nil
}

// Encrypt возвращает nonce||ciphertext. Пустой вход → nil.
func (c *Cipher) Encrypt(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plain, nil), nil
}

func (c *Cipher) EncryptString(s string) ([]byte, error) { return c.Encrypt([]byte(s)) }

// Decrypt разбирает nonce||ciphertext. Пустой вход → nil.
func (c *Cipher) Decrypt(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	ns := c.aead.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return c.aead.Open(nil, nonce, ct, nil)
}

func (c *Cipher) DecryptString(blob []byte) (string, error) {
	b, err := c.Decrypt(blob)
	return string(b), err
}

// HMAC — детерминированный hex HMAC-SHA256; тег для поиска/уникальности ИИН.
func (c *Cipher) HMAC(s string) string {
	m := hmac.New(sha256.New, c.hmacKey)
	m.Write([]byte(s))
	return hex.EncodeToString(m.Sum(nil))
}

package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedMFASecretPrefix = "enc:v1:"

type MFASecretCipher struct {
	aead cipher.AEAD
}

func NewMFASecretCipher(key string) *MFASecretCipher {
	derived := sha256.Sum256([]byte(key))
	block, _ := aes.NewCipher(derived[:])
	aead, _ := cipher.NewGCM(block)
	return &MFASecretCipher{aead: aead}
}

func (c *MFASecretCipher) Encrypt(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate mfa secret nonce: %w", err)
	}
	sealed := c.aead.Seal(nil, nonce, []byte(secret), nil)
	payload := append(nonce, sealed...)
	return encryptedMFASecretPrefix + base64.RawStdEncoding.EncodeToString(payload), nil
}

func (c *MFASecretCipher) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, encryptedMFASecretPrefix) {
		return stored, nil
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, encryptedMFASecretPrefix))
	if err != nil {
		return "", fmt.Errorf("decode mfa secret: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("invalid encrypted mfa secret")
	}
	plain, err := c.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt mfa secret: %w", err)
	}
	return string(plain), nil
}

// Package session protege o estado do navegador antes de persistir no banco.
package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

var ErrInvalid = errors.New("sessão criptografada inválida")

type Cipher struct{ aead cipher.AEAD }

func New(encodedKey string) (*Cipher, error) {
	key, err := base64.StdEncoding.Strict().DecodeString(encodedKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("SESSION_ENCRYPTION_KEY deve conter 32 bytes em base64")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalid
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalid
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Seal(connectionID string, state []byte) ([]byte, error) {
	if connectionID == "" || len(state) == 0 {
		return nil, ErrInvalid
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, ErrInvalid
	}
	envelope := append([]byte{1}, nonce...)
	return c.aead.Seal(envelope, nonce, state, []byte(connectionID)), nil
}

func (c *Cipher) Open(connectionID string, envelope []byte) ([]byte, error) {
	n := c.aead.NonceSize()
	if connectionID == "" || len(envelope) < 1+n+c.aead.Overhead() || envelope[0] != 1 {
		return nil, ErrInvalid
	}
	plain, err := c.aead.Open(nil, envelope[1:1+n], envelope[1+n:], []byte(connectionID))
	if err != nil {
		return nil, ErrInvalid
	}
	return plain, nil
}

// Package secretbox encrypts small secrets kept in the database (TOTP
// secrets), so that a copy of the database alone does not reveal them.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// ErrOpen means a sealed value was not made with this key, or was changed.
var ErrOpen = errors.New("cannot open sealed value")

// Box seals and opens values with AES-256-GCM.
type Box struct {
	aead cipher.AEAD
}

// New takes the key as base64 of 32 random bytes, e.g. from
// `openssl rand -base64 32`.
func New(keyBase64 string) (*Box, error) {

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("key is not base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key is %d bytes, want 32", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Box{aead: aead}, nil
}

// Seal returns a random nonce followed by the encrypted value.
func (b *Box) Seal(plaintext []byte) ([]byte, error) {

	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return b.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Open returns the value Seal sealed, or ErrOpen.
func (b *Box) Open(sealed []byte) ([]byte, error) {

	n := b.aead.NonceSize()
	if len(sealed) < n {
		return nil, ErrOpen
	}

	plaintext, err := b.aead.Open(nil, sealed[:n], sealed[n:], nil)
	if err != nil {
		return nil, ErrOpen
	}
	return plaintext, nil
}

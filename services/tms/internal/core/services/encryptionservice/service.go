package encryptionservice

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

var (
	ErrValueRequired    = errors.New("value is required")
	ErrCiphertextFormat = errors.New("invalid ciphertext format")
	ErrCiphertextShort  = errors.New("ciphertext too short")
)

type Params struct {
	fx.In

	Config *config.Config
}

type Service struct {
	key []byte
}

func New(p Params) *Service {
	trimmed := strings.TrimSpace(p.Config.Security.Encryption.Key)
	if trimmed == "" {
		return &Service{}
	}

	sum := sha256.Sum256([]byte(trimmed))
	return &Service{key: sum[:]}
}

func (s *Service) EncryptString(value string) (string, error) {
	plaintext := strings.TrimSpace(value)
	if plaintext == "" {
		return "", ErrValueRequired
	}
	if len(s.key) == 0 {
		return plaintext, nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *Service) DecryptString(value string) (string, error) {
	ciphertext := strings.TrimSpace(value)
	if ciphertext == "" {
		return "", ErrValueRequired
	}
	if len(s.key) == 0 {
		return ciphertext, nil
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrCiphertextFormat
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", ErrCiphertextShort
	}

	nonce, payload := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

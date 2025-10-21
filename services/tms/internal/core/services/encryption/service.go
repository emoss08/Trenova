package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrCiphertextTooShort = errors.New("ciphertext too short")

type Service interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

type service struct {
	l   *zap.Logger
	key []byte
}

func NewService(p ServiceParams) Service {
	log := p.Logger.Named("encryption-service")

	encryptionKey := p.Config.Security.Encryption.Key
	if encryptionKey == "" {
		log.Fatal("encryption key is required")
	}

	key := []byte(encryptionKey)
	if len(key) < 32 {
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		key = key[:32]
	}

	return &service{
		l:   log,
		key: key,
	}
}

func (s *service) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *service) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, ciphertextData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

package tokenutils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// New mints a single-use URL-safe token: 32 bytes of crypto randomness,
// base64url-encoded for the link, with the SHA-256 hex digest for storage so
// a database leak never exposes usable tokens.
func New() (token, tokenHash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, Hash(token), nil
}

// Hash derives the stored lookup digest for a raw token.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomHex returns size bytes of crypto randomness as lowercase hex.
func RandomHex(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("random hex size must be positive, got %d", size)
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random hex: %w", err)
	}

	return hex.EncodeToString(raw), nil
}

// RandomSeed returns 32 bytes of crypto randomness, sized for a ChaCha8 seed.
func RandomSeed() ([32]byte, error) {
	var seed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return seed, fmt.Errorf("generate random seed: %w", err)
	}

	return seed, nil
}

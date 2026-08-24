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

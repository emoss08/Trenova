package tokenutils

import (
	"crypto/sha256"
	"encoding/base64"
)

func PKCEVerifier() (string, error) {
	verifier, _, err := New()
	return verifier, err
}

func PKCEChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

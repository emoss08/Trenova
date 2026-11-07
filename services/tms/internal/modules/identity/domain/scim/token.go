package scim

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const (
	TokenPrefix        = "scim_"
	tokenSeparator     = "."
	maxPresentedLength = 256
)

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func TokenLookupPrefix(token string) (string, bool) {
	if len(token) > maxPresentedLength {
		return "", false
	}

	prefix, secret, found := strings.Cut(token, tokenSeparator)
	if !found || secret == "" || !strings.HasPrefix(prefix, TokenPrefix) ||
		len(prefix) <= len(TokenPrefix) {
		return "", false
	}

	return prefix, true
}

func JoinToken(prefix, secret string) string {
	return prefix + tokenSeparator + secret
}

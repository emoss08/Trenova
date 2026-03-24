package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var errInvalidToken = errors.New("invalid token format")

func SplitToken(token string) (string, string, error) {
	trimmed := strings.TrimSpace(token)
	parts := strings.SplitN(trimmed, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errInvalidToken
	}
	return parts[0], parts[1], nil
}

func HashSecret(salt, secret string) string {
	sum := sha256.Sum256([]byte(salt + ":" + secret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type GeneratedSecret struct {
	Prefix string
	Secret string
	Hash   string
	Salt   string
}

func (g *GeneratedSecret) Token() string {
	return g.Prefix + "." + g.Secret
}

func GenerateAPIKeySecret() (*GeneratedSecret, error) {
	prefixBytes := make([]byte, 9)
	secretBytes := make([]byte, 24)
	saltBytes := make([]byte, 18)
	if _, err := rand.Read(prefixBytes); err != nil {
		return nil, err
	}
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, err
	}
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, err
	}

	prefix := "trv_" + base64.RawURLEncoding.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	salt := base64.RawURLEncoding.EncodeToString(saltBytes)
	return &GeneratedSecret{
		Prefix: prefix,
		Secret: secret,
		Hash:   HashSecret(salt, secret),
		Salt:   salt,
	}, nil
}

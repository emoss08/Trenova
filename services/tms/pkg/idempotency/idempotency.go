package idempotency

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	HeaderKey      = "Idempotency-Key"
	HeaderReplayed = "Idempotent-Replayed"
	MaxKeyLength   = 255
	ScopedKeyLen   = 64

	keyDerivationLabel = "trenova/idempotency/v1"
)

type contextKey struct{}

type Scope struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	PrincipalType  string
	PrincipalID    pulid.ID
}

type Hasher struct {
	key []byte
}

func NewHasher(secret string) Hasher {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, keyDerivationLabel)
	return Hasher{key: mac.Sum(nil)}
}

func (h Hasher) ScopedKey(scope Scope, clientKey string) string {
	mac := h.mac()
	writeParts(
		mac,
		scope.OrganizationID.String(),
		scope.BusinessUnitID.String(),
		scope.PrincipalType,
		scope.PrincipalID.String(),
		clientKey,
	)
	return hex.EncodeToString(mac.Sum(nil))
}

func (h Hasher) Fingerprint(method, target string, body []byte) string {
	mac := h.mac()
	writeParts(mac, method, target)
	_, _ = mac.Write(separator)
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func (h Hasher) mac() hash.Hash {
	return hmac.New(sha256.New, h.key)
}

func WithKey(ctx context.Context, scopedKey string) context.Context {
	if scopedKey == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, scopedKey)
}

func KeyFrom(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	key, ok := ctx.Value(contextKey{}).(string)
	return key, ok && key != ""
}

func ValidClientKey(key string) bool {
	if key == "" || len(key) > MaxKeyLength {
		return false
	}
	for i := range len(key) {
		if key[i] < 0x21 || key[i] > 0x7e {
			return false
		}
	}
	return true
}

var separator = []byte{0}

func writeParts(w io.Writer, parts ...string) {
	for i, part := range parts {
		if i > 0 {
			_, _ = w.Write(separator)
		}
		_, _ = io.WriteString(w, part)
	}
}

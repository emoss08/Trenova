package idempotency

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"

	"github.com/emoss08/trenova/shared/hashutils"
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
	return Hasher{key: hashutils.DeriveKey(secret, keyDerivationLabel)}
}

func (h Hasher) ScopedKey(scope Scope, clientKey string) string {
	mac := h.mac()
	hashutils.WriteParts(
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
	hashutils.WriteParts(mac, method, target)
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

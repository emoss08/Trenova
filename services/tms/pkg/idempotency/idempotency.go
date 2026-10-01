package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	HeaderKey      = "Idempotency-Key"
	HeaderReplayed = "Idempotent-Replayed"
	MaxKeyLength   = 255
	ScopedKeyLen   = 64
)

type contextKey struct{}

type Scope struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	PrincipalType  string
	PrincipalID    pulid.ID
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

func ScopedKey(scope Scope, clientKey string) string {
	return digest(
		scope.OrganizationID.String(),
		scope.BusinessUnitID.String(),
		scope.PrincipalType,
		scope.PrincipalID.String(),
		clientKey,
	)
}

func Fingerprint(method, target string, body []byte) string {
	h := sha256.New()
	_, _ = io.WriteString(h, method)
	_, _ = h.Write(separator)
	_, _ = io.WriteString(h, target)
	_, _ = h.Write(separator)
	_, _ = h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

var separator = []byte{0}

func digest(parts ...string) string {
	h := sha256.New()
	for i, part := range parts {
		if i > 0 {
			_, _ = h.Write(separator)
		}
		_, _ = io.WriteString(h, part)
	}
	return hex.EncodeToString(h.Sum(nil))
}

package postgres

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/dbscope"
)

const (
	scopeTokenVersion = "v1"
	scopeNoUser       = "-"
	minScopeKeyBytes  = sha256.Size
	hmacBlockSize     = sha256.BlockSize
	innerPadByte      = 0x36
	outerPadByte      = 0x5c
)

var (
	ErrInvalidScopeKey      = errors.New("row-level security scope key is invalid")
	ErrInvalidTenantScope   = errors.New("tenant scope is missing an organization or business unit")
	ErrMalformedTenantScope = errors.New("tenant scope contains characters that cannot be signed")

	scopeKeyIDPattern    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	scopeIdentPattern    = regexp.MustCompile(`^[A-Za-z0-9_]{1,100}$`)
	scopeTokenCharacters = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
)

type scopeSigner struct {
	keyID string
	key   []byte
	ttl   time.Duration
	now   func() time.Time
}

type scopePads struct {
	Inner []byte
	Outer []byte
}

func newScopeSigner(keyID string, key []byte, ttl time.Duration) (*scopeSigner, error) {
	if !scopeKeyIDPattern.MatchString(keyID) || len(key) < minScopeKeyBytes || ttl <= 0 {
		return nil, ErrInvalidScopeKey
	}

	return &scopeSigner{
		keyID: keyID,
		key:   append([]byte(nil), key...),
		ttl:   ttl,
		now:   time.Now,
	}, nil
}

func (s *scopeSigner) token(tenant dbscope.Tenant) (string, error) {
	if !tenant.Valid() {
		return "", ErrInvalidTenantScope
	}

	org := tenant.OrganizationID.String()
	bu := tenant.BusinessUnitID.String()
	user := scopeNoUser
	if !tenant.UserID.IsNil() {
		user = tenant.UserID.String()
	}

	if !scopeIdentPattern.MatchString(org) || !scopeIdentPattern.MatchString(bu) ||
		(user != scopeNoUser && !scopeIdentPattern.MatchString(user)) {
		return "", ErrMalformedTenantScope
	}

	expires := strconv.FormatInt(s.now().Add(s.ttl).Unix(), 10)

	var b strings.Builder
	b.Grow(len(scopeTokenVersion) + len(s.keyID) + len(org) + len(bu) + len(user) + len(expires) + 6 + hex.EncodedLen(sha256.Size))
	b.WriteString(scopeTokenVersion)
	b.WriteByte('.')
	b.WriteString(s.keyID)
	b.WriteByte('.')
	b.WriteString(org)
	b.WriteByte('.')
	b.WriteString(bu)
	b.WriteByte('.')
	b.WriteString(user)
	b.WriteByte('.')
	b.WriteString(expires)

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(b.String()))
	b.WriteByte('.')
	b.WriteString(hex.EncodeToString(mac.Sum(nil)))

	token := b.String()
	if !scopeTokenCharacters.MatchString(token) {
		return "", ErrMalformedTenantScope
	}

	return token, nil
}

func (s *scopeSigner) pads() scopePads {
	return deriveScopePads(s.key)
}

func deriveScopePads(key []byte) scopePads {
	normalized := make([]byte, hmacBlockSize)
	if len(key) > hmacBlockSize {
		sum := sha256.Sum256(key)
		copy(normalized, sum[:])
	} else {
		copy(normalized, key)
	}

	pads := scopePads{
		Inner: make([]byte, hmacBlockSize),
		Outer: make([]byte, hmacBlockSize),
	}
	for i, b := range normalized {
		pads.Inner[i] = b ^ innerPadByte
		pads.Outer[i] = b ^ outerPadByte
	}

	return pads
}

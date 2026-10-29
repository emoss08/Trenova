package aiproviderkey

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// RotationGrace is how long a replaced key is kept as a fallback when its
	// replacement is saved with the old one kept.
	RotationGrace = 24 * time.Hour
)

// KeyInfo describes the stored credential without the secret. It is nil
// when no key is stored.
type KeyInfo struct {
	Prefix               string
	LastFour             string
	AddedAt              int64
	AddedByID            pulid.ID
	LastUsedAt           *int64
	PreviousKeyExpiresAt *int64
}

// Describe is what may be said about the stored key. A key stored before its
// ends were recorded reads as added when the provider was created.
func Describe(p *aiprovider.Provider, now int64) *KeyInfo {
	if !p.HasStoredAPIKey() && !p.HasAPIKey {
		return nil
	}

	info := &KeyInfo{
		Prefix:     p.APIKeyPrefix,
		LastFour:   p.APIKeyLastFour,
		AddedAt:    p.CreatedAt,
		AddedByID:  p.APIKeyAddedByID,
		LastUsedAt: p.APIKeyLastUsedAt,
	}
	if p.APIKeyAddedAt != nil {
		info.AddedAt = *p.APIKeyAddedAt
	}
	if p.RotationExpiresAt != nil && *p.RotationExpiresAt > now {
		expires := *p.RotationExpiresAt
		info.PreviousKeyExpiresAt = &expires
	}

	return info
}

// NeedsKeyFingerprint reports a stored key whose ends were never recorded.
func NeedsKeyFingerprint(p *aiprovider.Provider) bool {
	return p.HasStoredAPIKey() && p.APIKeyLastFour == "" && p.APIKeyPrefix == ""
}

// PreviousKeyUsable reports a replaced key still inside its grace period.
func PreviousKeyUsable(p *aiprovider.Provider, now int64) bool {
	return strings.TrimSpace(p.PreviousAPIKey) != "" &&
		p.RotationExpiresAt != nil && *p.RotationExpiresAt > now
}

// ForgetExpiredPreviousKey drops a replaced key whose grace period is over,
// and reports whether there was one.
func ForgetExpiredPreviousKey(p *aiprovider.Provider, now int64) bool {
	if !p.PreviousKeyExpired(now) {
		return false
	}
	p.PreviousAPIKey = ""
	p.RotationExpiresAt = nil

	return true
}

// KeyReplacement is a new credential for a provider. Encrypted is empty to
// clear the stored key.
type KeyReplacement struct {
	Encrypted    string
	Prefix       string
	LastFour     string
	AddedByID    pulid.ID
	Now          int64
	KeepPrevious bool
}

// ReplaceAPIKey stores a new credential. With KeepPrevious, a key it
// replaces is kept as a fallback for RotationGrace; otherwise any replaced
// key is dropped.
func ReplaceAPIKey(p *aiprovider.Provider, r *KeyReplacement) {
	if r.KeepPrevious && r.Encrypted != "" && p.HasStoredAPIKey() {
		expires := r.Now + int64(RotationGrace/time.Second)
		p.PreviousAPIKey = p.APIKey
		p.RotationExpiresAt = &expires
	} else {
		p.PreviousAPIKey = ""
		p.RotationExpiresAt = nil
	}

	p.APIKey = r.Encrypted
	p.APIKeyLastUsedAt = nil
	if r.Encrypted == "" {
		p.APIKeyPrefix = ""
		p.APIKeyLastFour = ""
		p.APIKeyAddedAt = nil
		p.APIKeyAddedByID = pulid.Nil

		return
	}

	addedAt := r.Now
	p.APIKeyPrefix = r.Prefix
	p.APIKeyLastFour = r.LastFour
	p.APIKeyAddedAt = &addedAt
	p.APIKeyAddedByID = r.AddedByID
}

package supportaccess

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/keel"
)

const (
	CurrentKeySetID = "current"
	MaxKeys         = 4
	keyTypeOKP      = "OKP"
	curveEd25519    = "Ed25519"
	algorithmEdDSA  = "EdDSA"
	useSignature    = "sig"
	maxInstanceID   = 100
)

var (
	keyIDPattern      = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	instanceIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)
)

// SigningKey is one of the control plane's support-signing public keys as a JSON Web Key.
type SigningKey struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Curve     string `json:"crv"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
	X         string `json:"x"`
}

func (k SigningKey) PublicKey() (ed25519.PublicKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("key %q: x is not unpadded base64url", k.KeyID)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("key %q: x is not a 32-byte Ed25519 public key", k.KeyID)
	}
	return ed25519.PublicKey(raw), nil
}

func (k SigningKey) validate() error {
	if !keyIDPattern.MatchString(k.KeyID) {
		return errors.New("every key needs a kid of 1 to 64 letters, digits, dots, dashes or underscores")
	}
	if k.KeyType != keyTypeOKP || k.Curve != curveEd25519 || k.Algorithm != algorithmEdDSA {
		return fmt.Errorf("key %q must be kty OKP, crv Ed25519, alg EdDSA", k.KeyID)
	}
	if k.Use != useSignature {
		return fmt.Errorf("key %q must have use sig", k.KeyID)
	}
	_, err := k.PublicKey()
	return err
}

// KeySet is what the control plane, through the operator, tells the instance about support
// access: which keys sign assertions, which instance this is (the assertions' audience), and
// the Trenova staff email domain grants are limited to.
type KeySet struct {
	keel.Table `keel:"managed_support_access_states"`

	ID          string       `json:"id"          keel:"id,pk"`
	InstanceID  string       `json:"instanceId"  keel:"instance_id,nullzero"`
	StaffDomain string       `json:"staffDomain" keel:"staff_domain"`
	Keys        []SigningKey `json:"keys"        keel:"keys,jsonb"`
	ReceivedAt  int64        `json:"receivedAt"  keel:"received_at"`
}

// Normalize checks a delivered key set and returns it in its stored form. A delivery with no
// key is refused rather than stored, so a broken delivery cannot silently replace working keys;
// keys are retired by delivering a set without them.
func (ks *KeySet) Normalize(shared bool) error {
	ks.InstanceID = strings.TrimSpace(ks.InstanceID)
	switch {
	case shared && ks.InstanceID != "":
		return errors.New(
			"instanceId must be left out on a shared server: each assertion names its tenant's instance",
		)
	case !shared && (!instanceIDPattern.MatchString(ks.InstanceID) ||
		len(ks.InstanceID) > maxInstanceID):
		return errors.New("instanceId must be 1 to 100 letters, digits, dots, colons, dashes or underscores")
	}
	ks.StaffDomain = lowerASCII(strings.TrimSpace(ks.StaffDomain))
	if !ValidDomain(ks.StaffDomain) {
		return errors.New("staffDomain must be a lower-case ASCII domain name")
	}
	if len(ks.Keys) == 0 || len(ks.Keys) > MaxKeys {
		return fmt.Errorf("keys must hold between 1 and %d keys", MaxKeys)
	}
	seen := make(map[string]struct{}, len(ks.Keys))
	for _, key := range ks.Keys {
		if err := key.validate(); err != nil {
			return err
		}
		if _, dup := seen[key.KeyID]; dup {
			return fmt.Errorf("key %q is listed twice", key.KeyID)
		}
		seen[key.KeyID] = struct{}{}
	}
	slices.SortFunc(ks.Keys, func(a, b SigningKey) int { return strings.Compare(a.KeyID, b.KeyID) })
	ks.ID = CurrentKeySetID
	return nil
}

// Key returns the public key with the given id.
func (ks *KeySet) Key(keyID string) (ed25519.PublicKey, bool) {
	if ks == nil {
		return nil, false
	}
	for _, key := range ks.Keys {
		if key.KeyID == keyID {
			pub, err := key.PublicKey()
			if err != nil {
				return nil, false
			}
			return pub, true
		}
	}
	return nil, false
}

// Same reports whether two key sets say the same thing, ignoring when they arrived.
func (ks *KeySet) Same(other *KeySet) bool {
	if ks == nil || other == nil {
		return ks == other
	}
	return ks.InstanceID == other.InstanceID &&
		ks.StaffDomain == other.StaffDomain &&
		slices.Equal(ks.Keys, other.Keys)
}

// AssertionUse records one accepted assertion id until it could no longer be valid.
type AssertionUse struct {
	keel.Table `keel:"support_assertion_uses"`

	AssertionID string `json:"assertionId" keel:"assertion_id,pk"`
	UsedAt      int64  `json:"usedAt"      keel:"used_at"`
	ExpiresAt   int64  `json:"expiresAt"   keel:"expires_at"`
}

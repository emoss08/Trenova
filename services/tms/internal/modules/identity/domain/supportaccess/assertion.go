package supportaccess

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	jose "github.com/go-jose/go-jose/v4"
)

// The support assertion is the only way a Trenova staff member enters an instance. The control
// plane signs it, for one staff member, one instance and one organization, to be used once
// within a minute. docs/engineering/managed-mode.md ("Support access") is the contract.
const (
	AssertionIssuer      = "trenova-control-plane"
	AssertionType        = "support-assertion+jwt"
	MaxAssertionBytes    = 4096
	MaxAssertionLifetime = 60 * time.Second
	AssertionClockSkew   = 15 * time.Second
	maxStaffNameRunes    = 100
)

var (
	assertionIDPattern    = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)
	organizationIDPattern = regexp.MustCompile(`^org_[0-9A-HJKMNP-TV-Z]{26}$`)
)

// RefusalReason says why an assertion or an entry was refused. It goes to the security log and
// the instance's logs only: the caller is told nothing but that entry was refused.
type RefusalReason string

const (
	RefusalMalformed      = RefusalReason("malformed")
	RefusalAlgorithm      = RefusalReason("algorithm")
	RefusalHeader         = RefusalReason("header")
	RefusalUnknownKey     = RefusalReason("unknown_key")
	RefusalSignature      = RefusalReason("signature")
	RefusalClaims         = RefusalReason("claims")
	RefusalIssuer         = RefusalReason("issuer")
	RefusalAudience       = RefusalReason("audience")
	RefusalExpired        = RefusalReason("expired")
	RefusalNotYetValid    = RefusalReason("not_yet_valid")
	RefusalLifetime       = RefusalReason("lifetime")
	RefusalStaffEmail     = RefusalReason("staff_email")
	RefusalReplayed       = RefusalReason("replayed")
	RefusalNoGrant        = RefusalReason("no_live_grant")
	RefusalNotConfigured  = RefusalReason("not_configured")
	RefusalRateLimited    = RefusalReason("rate_limited")
	RefusalOrganization   = RefusalReason("organization")
	RefusalTenant         = RefusalReason("tenant")
	RefusalSessionFailure = RefusalReason("session_failure")
)

// RefusalError carries the reason an entry was refused. Its Error never reaches the caller.
type RefusalError struct {
	Reason RefusalReason
	Detail string
}

func (e *RefusalError) Error() string {
	if e.Detail == "" {
		return "support entry refused: " + string(e.Reason)
	}
	return "support entry refused: " + string(e.Reason) + ": " + e.Detail
}

func refuse(reason RefusalReason, detail string) error {
	return &RefusalError{Reason: reason, Detail: detail}
}

// ReasonOf returns the refusal reason of err, or RefusalSessionFailure for anything else.
func ReasonOf(err error) RefusalReason {
	var refusal *RefusalError
	if errors.As(err, &refusal) {
		return refusal.Reason
	}
	return RefusalSessionFailure
}

// Assertion is a verified support assertion.
type Assertion struct {
	ID             string
	Audience       string
	StaffEmail     string
	StaffName      string
	OrganizationID pulid.ID
	IssuedAt       int64
	ExpiresAt      int64
}

type assertionClaims struct {
	Issuer       string `json:"iss"`
	Subject      string `json:"sub"`
	Audience     string `json:"aud"`
	ID           string `json:"jti"`
	IssuedAt     int64  `json:"iat"`
	NotBefore    int64  `json:"nbf"`
	ExpiresAt    int64  `json:"exp"`
	Name         string `json:"name"`
	Organization string `json:"org"`
}

var strictClaims = sonic.Config{DisallowUnknownFields: true}.Froze()

// VerifyAssertion checks a support assertion against the delivered keys at now. It accepts
// only a compact JWS signed with EdDSA by a key the control plane delivered, of the support
// assertion type, for this instance, unexpired and issued no more than a minute before it
// expires. Replay is the caller's to refuse: it records the id once the rest has passed.
func VerifyAssertion(token string, keys *KeySet, now time.Time) (*Assertion, error) {
	if keys == nil || len(keys.Keys) == 0 || keys.InstanceID == "" {
		return nil, refuse(RefusalNotConfigured, "")
	}
	return verifyAssertion(token, keys, now, false)
}

func VerifyTenantAssertion(token string, keys *KeySet, now time.Time) (*Assertion, error) {
	if keys == nil || len(keys.Keys) == 0 || keys.InstanceID != "" {
		return nil, refuse(RefusalNotConfigured, "")
	}
	return verifyAssertion(token, keys, now, true)
}

func verifyAssertion(token string, keys *KeySet, now time.Time, tenant bool) (*Assertion, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > MaxAssertionBytes || strings.Count(token, ".") != 2 {
		return nil, refuse(RefusalMalformed, "")
	}

	jws, err := jose.ParseSignedCompact(token, []jose.SignatureAlgorithm{jose.EdDSA})
	if err != nil {
		return nil, refuse(RefusalAlgorithm, err.Error())
	}
	if len(jws.Signatures) != 1 {
		return nil, refuse(RefusalHeader, "not exactly one signature")
	}
	header := jws.Signatures[0].Protected
	if header.Algorithm != string(jose.EdDSA) {
		return nil, refuse(RefusalAlgorithm, header.Algorithm)
	}
	if header.JSONWebKey != nil || header.KeyID == "" {
		return nil, refuse(RefusalHeader, "an embedded key or no key id")
	}
	if typ, _ := header.ExtraHeaders[jose.HeaderType].(string); typ != AssertionType {
		return nil, refuse(RefusalHeader, "type")
	}

	pub, ok := keys.Key(header.KeyID)
	if !ok {
		return nil, refuse(RefusalUnknownKey, header.KeyID)
	}
	payload, err := jws.Verify(pub)
	if err != nil {
		return nil, refuse(RefusalSignature, "")
	}

	var claims assertionClaims
	if err = strictClaims.Unmarshal(payload, &claims); err != nil {
		return nil, refuse(RefusalClaims, "the claims are not a support assertion")
	}

	return claims.verify(keys, now, tenant)
}

func (c *assertionClaims) verify(keys *KeySet, now time.Time, tenant bool) (*Assertion, error) {
	if c.Issuer != AssertionIssuer {
		return nil, refuse(RefusalIssuer, c.Issuer)
	}
	if !tenant && (c.Audience == "" || c.Audience != keys.InstanceID) {
		return nil, refuse(RefusalAudience, c.Audience)
	}
	if !assertionIDPattern.MatchString(c.ID) {
		return nil, refuse(RefusalClaims, "jti")
	}

	at := now.Unix()
	skew := int64(AssertionClockSkew / time.Second)
	if c.IssuedAt <= 0 || c.ExpiresAt <= c.IssuedAt {
		return nil, refuse(RefusalLifetime, "")
	}
	if c.ExpiresAt-c.IssuedAt > int64(MaxAssertionLifetime/time.Second) {
		return nil, refuse(RefusalLifetime, "")
	}
	if c.IssuedAt > at+skew || (c.NotBefore != 0 && c.NotBefore > at+skew) {
		return nil, refuse(RefusalNotYetValid, "")
	}
	if c.ExpiresAt+skew <= at {
		return nil, refuse(RefusalExpired, "")
	}

	email, err := NormalizeStaffEmail(c.Subject, keys.StaffDomain)
	if err != nil {
		return nil, refuse(RefusalStaffEmail, err.Error())
	}
	name, ok := staffName(c.Name)
	if !ok {
		return nil, refuse(RefusalClaims, "name")
	}
	if !organizationIDPattern.MatchString(c.Organization) {
		return nil, refuse(RefusalClaims, "org")
	}
	orgID := pulid.ID(c.Organization)

	return &Assertion{
		ID:             c.ID,
		Audience:       c.Audience,
		StaffEmail:     email,
		StaffName:      name,
		OrganizationID: orgID,
		IssuedAt:       c.IssuedAt,
		ExpiresAt:      c.ExpiresAt,
	}, nil
}

func staffName(raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxStaffNameRunes {
		return "", false
	}
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", false
		}
	}
	return name, true
}

// UseExpiresAt is how long an accepted assertion's id is kept: past the latest moment the
// assertion could still verify.
func (a *Assertion) UseExpiresAt() int64 {
	return a.ExpiresAt + int64(AssertionClockSkew/time.Second)
}

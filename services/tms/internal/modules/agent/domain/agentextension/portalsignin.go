package agentextension

import (
	"encoding/base32"
	"errors"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
)

const (
	PortalCredentialAuthenticator = "authenticatorKey"
	// PortalSecretAuthenticator is the placeholder the browser types for a
	// sign-in code. Browser Use generates a fresh code from the authenticator
	// key whenever it types a secret named with this suffix, so the key, not
	// a code, is what is stored and sent.
	PortalSecretAuthenticator = "portal_bu_2fa_code"

	// PortalSignInCodeMarker is what the browser is told to begin its answer
	// with when the portal asks for a code it was not given.
	PortalSignInCodeMarker = "SIGN-IN CODE NEEDED"

	minAuthenticatorKeyBytes = 10
)

var (
	ErrAuthenticatorKeyInvalid = errors.New(
		"the authenticator key must be the setup key the portal showed for an authenticator app",
	)

	portalSecretPlaceholders = []string{
		PortalSecretUsername,
		PortalSecretPassword,
		PortalSecretAuthenticator,
	}

	// signInCodeWords are how a browser that did not use the marker still
	// says the portal stopped it for a code.
	signInCodeWords = []string{
		"2fa", "two-factor", "two factor", "multi-factor", "mfa", "verification code",
		"one-time code", "one-time password", "authenticator", "security code", "passcode",
	}
)

func authenticatorField() configspec.Field {
	return configspec.Field{
		Key:       PortalCredentialAuthenticator,
		Label:     "Authenticator key",
		Type:      configspec.FieldTypePassword,
		Sensitive: true,
	}
}

func (s *PortalSite) HasAuthenticator() bool {
	return s != nil && configspec.HasRequired(s.Credentials, []configspec.Field{
		{Key: PortalCredentialAuthenticator, Required: true},
	})
}

// NormalizeAuthenticatorKey reads the key a portal shows when an
// authenticator app is set up, as a person copies it: grouped with spaces or
// dashes, in either case, or as the otpauth:// link a QR code holds. Blank is
// no key; anything that is not a base32 key of a usable length is refused.
func NormalizeAuthenticatorKey(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(strings.ToLower(value), "otpauth://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", ErrAuthenticatorKeyInvalid
		}
		value = parsed.Query().Get("secret")
	}

	key := strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "\t", "").Replace(value))
	key = strings.TrimRight(key, "=")
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(key)
	if err != nil || len(decoded) < minAuthenticatorKeyBytes {
		return "", ErrAuthenticatorKeyInvalid
	}

	return key, nil
}

// AsksForSignInCode says the browser stopped because the portal wanted a
// code: it began its answer with the marker, or a task that did not succeed
// talks about a code. A finished lookup that merely mentions two-factor
// settings is not one.
func AsksForSignInCode(answer string, succeeded bool) bool {
	text := strings.ToLower(answer)
	if strings.Contains(text, strings.ToLower(PortalSignInCodeMarker)) {
		return true
	}
	if succeeded {
		return false
	}
	for _, word := range signInCodeWords {
		if strings.Contains(text, word) {
			return true
		}
	}

	return false
}

// namesSecret is a detail named like a placeholder the browser would read
// as a sign-in secret.
func namesSecret(name string) bool {
	lower := strings.ToLower(name)
	for _, placeholder := range portalSecretPlaceholders {
		if strings.Contains(lower, placeholder) {
			return true
		}
	}

	return strings.Contains(lower, "bu_2fa_code")
}

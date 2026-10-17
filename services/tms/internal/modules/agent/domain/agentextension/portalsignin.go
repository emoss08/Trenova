package agentextension

import (
	"encoding/base32"
	"errors"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/shared/stringutils"
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
		PortalSecretSignInCode,
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

const (
	// PortalSecretSignInCode is the placeholder a follow-up task types for a
	// code a person relayed; the code itself is sent only as a vendor secret.
	PortalSecretSignInCode = "portal_sign_in_code"

	// PortalCodeWaitSeconds is how long a browser is kept open on a code
	// prompt for a person to enter the code.
	PortalCodeWaitSeconds    = 10 * 60
	MaxPortalCodeAttempts    = 3
	MaxPortalCodePromptRunes = 500
	minSignInCodeLength      = 4
	maxSignInCodeLength      = 12
)

var ErrSignInCodeInvalid = errors.New(
	"a sign-in code is 4 to 12 letters or digits, as the portal sent it",
)

// NormalizeSignInCode reads a code as a person types it from a text message
// or an email: grouped with spaces or dashes, which portals ignore. Anything
// that is not 4 to 12 letters and digits is refused, so nothing else rides
// into the portal with it.
func NormalizeSignInCode(raw string) (string, error) {
	code := strings.NewReplacer(" ", "", "-", "", "\t", "").Replace(strings.TrimSpace(raw))
	if len(code) < minSignInCodeLength || len(code) > maxSignInCodeLength {
		return "", ErrSignInCodeInvalid
	}
	for _, char := range code {
		isDigit := char >= '0' && char <= '9'
		isLetter := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
		if !isDigit && !isLetter {
			return "", ErrSignInCodeInvalid
		}
	}

	return code, nil
}

// ContinueWithCode is the follow-up task that carries on in the same browser
// once a person relayed the code: enter it where the portal asked, then do
// what the first task was given, without repeating what it already did.
func ContinueWithCode(task string) string {
	return "The portal is showing where to enter the sign-in code it sent. Enter " +
		PortalSecretSignInCode + " there and submit it. Then carry on with the task below from " +
		"where you stopped, without repeating anything already done. If the portal says the " +
		"code is wrong or has expired, stop there and begin your answer with " +
		PortalSignInCodeMarker + ", then say what it said.\n\n" + task
}

// CodeExpiresAt is when a run waiting on a sign-in code stops waiting.
func (r *PortalRun) CodeExpiresAt() *int64 {
	if r == nil || r.Status != PortalRunStatusAwaitingCode || r.CodeRequestedAt == nil {
		return nil
	}
	expires := *r.CodeRequestedAt + PortalCodeWaitSeconds

	return &expires
}

// AwaitCode keeps a run open on the portal's code prompt. What it has done so
// far (the pages, steps and cost the outcome carries) is kept, so the task the
// code starts adds to it, and how the portal asked is kept for the person.
func (r *PortalRun) AwaitCode(progress *PortalRunOutcome, at int64) {
	r.Status = PortalRunStatusAwaitingCode
	r.CodePrompt = stringutils.TruncateRunes(strings.TrimSpace(progress.Answer),
		MaxPortalCodePromptRunes)
	r.VisitedURLs = boundedURLs(progress.VisitedURLs)
	r.OffSiteURLs = boundedURLs(progress.OffSiteURLs)
	r.Steps = progress.Steps
	r.CostUSD = progress.CostUSD
	requested := at
	r.CodeRequestedAt = &requested
}

// Continued says the run's current task was started by a relayed code, so
// its pages, steps and cost add to what the run already holds.
func (r *PortalRun) Continued() bool {
	return r != nil && r.CodeAttempts > 0
}

// ContinuedWith records the follow-up task a relayed code started.
func (r *PortalRun) ContinuedWith(taskID string) {
	r.Status = PortalRunStatusRunning
	r.VendorTaskID = taskID
	r.CodeAttempts++
	r.CodePrompt = ""
	r.CodeRequestedAt = nil
}

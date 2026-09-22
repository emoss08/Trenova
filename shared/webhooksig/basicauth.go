package webhooksig

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

// ParseBasicCredentials splits a "user:password" pair. Both halves must be
// present: an empty user or password is a credential anybody can guess. The
// password may itself contain colons; only the first one separates.
func ParseBasicCredentials(credentials string) (user, password string, ok bool) {
	user, password, found := strings.Cut(credentials, ":")
	if !found || user == "" || password == "" {
		return "", "", false
	}

	return user, password, true
}

// VerifyBasicAuth checks an Authorization header against configured
// "user:password" credentials.
//
// Postmark does not sign inbound webhooks; basic auth on the webhook URL is
// the protection it offers, and it sends the credentials back on every
// delivery. Both halves are compared in constant time and neither comparison
// short-circuits the other, so the response time says nothing about which
// half was wrong or how much of it matched.
func VerifyBasicAuth(expected, header string) error {
	wantUser, wantPassword, ok := ParseBasicCredentials(expected)
	if !ok || header == "" {
		return ErrMissingHeaders
	}

	scheme, encoded, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Basic") {
		return ErrNoMatch
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return ErrNoMatch
	}

	gotUser, gotPassword, found := strings.Cut(string(decoded), ":")
	if !found {
		return ErrNoMatch
	}

	userMatch := subtle.ConstantTimeCompare([]byte(gotUser), []byte(wantUser))
	passwordMatch := subtle.ConstantTimeCompare([]byte(gotPassword), []byte(wantPassword))
	if userMatch&passwordMatch != 1 {
		return ErrNoMatch
	}

	return nil
}

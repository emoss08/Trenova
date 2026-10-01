package sessiontoken

import (
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
)

const separator = "."

var ErrMalformed = errors.New("session token is malformed")

type Issued struct {
	Token      string
	SecretHash string
}

func Issue(sessionID pulid.ID) (*Issued, error) {
	secret, hash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	return &Issued{
		Token:      sessionID.String() + separator + secret,
		SecretHash: hash,
	}, nil
}

func Parse(token string) (pulid.ID, string, error) {
	rawID, secret, ok := strings.Cut(strings.TrimSpace(token), separator)
	if !ok || rawID == "" || secret == "" || strings.Contains(secret, separator) {
		return pulid.Nil, "", ErrMalformed
	}

	sessionID, err := pulid.Parse(rawID)
	if err != nil {
		return pulid.Nil, "", ErrMalformed
	}

	return sessionID, secret, nil
}

func Matches(secret, storedHash string) bool {
	if secret == "" || storedHash == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(tokenutils.Hash(secret)), []byte(storedHash)) == 1
}

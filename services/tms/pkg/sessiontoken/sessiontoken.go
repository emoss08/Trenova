package sessiontoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
)

const (
	separator          = "."
	keyDerivationLabel = "trenova session token key v1"
)

var ErrMalformed = errors.New("session token is malformed")

type Issued struct {
	Token      string
	SecretHash string
}

type Issuer struct {
	key []byte
}

func NewIssuer(secret string) *Issuer {
	return &Issuer{key: hashutils.DeriveKey(secret, keyDerivationLabel)}
}

func (i *Issuer) Issue(sessionID, userID pulid.ID) (*Issued, error) {
	secret, _, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	return &Issued{
		Token:      sessionID.String() + separator + secret,
		SecretHash: i.hash(sessionID, userID, secret),
	}, nil
}

func (i *Issuer) Matches(sessionID, userID pulid.ID, secret, storedHash string) bool {
	if secret == "" || storedHash == "" {
		return false
	}

	return hmac.Equal([]byte(i.hash(sessionID, userID, secret)), []byte(storedHash))
}

func (i *Issuer) hash(sessionID, userID pulid.ID, secret string) string {
	mac := hmac.New(sha256.New, i.key)
	hashutils.WriteParts(mac, sessionID.String(), userID.String(), secret)
	return hex.EncodeToString(mac.Sum(nil))
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

package supportaccess

import (
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
)

const (
	tokenVersion   = "v1"
	tokenSeparator = "."
	tokenParts     = 5
)

var ErrMalformedToken = errors.New("support session token is malformed")

type Token struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	SessionID      pulid.ID
	Secret         string
}

type IssuedToken struct {
	Value      string
	SecretHash string
}

func IssueToken(organizationID, businessUnitID, sessionID pulid.ID) (*IssuedToken, error) {
	secret, hash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	return &IssuedToken{
		Value: strings.Join([]string{
			tokenVersion,
			organizationID.String(),
			businessUnitID.String(),
			sessionID.String(),
			secret,
		}, tokenSeparator),
		SecretHash: hash,
	}, nil
}

func ParseToken(value string) (*Token, error) {
	parts := strings.Split(strings.TrimSpace(value), tokenSeparator)
	if len(parts) != tokenParts || parts[0] != tokenVersion || parts[4] == "" {
		return nil, ErrMalformedToken
	}

	ids := make([]pulid.ID, 0, 3)
	for _, raw := range parts[1:4] {
		id, err := pulid.Parse(raw)
		if err != nil || id.IsNil() {
			return nil, ErrMalformedToken
		}
		ids = append(ids, id)
	}

	return &Token{
		OrganizationID: ids[0],
		BusinessUnitID: ids[1],
		SessionID:      ids[2],
		Secret:         parts[4],
	}, nil
}

func (t *Token) Matches(secretHash string) bool {
	if t == nil || t.Secret == "" || secretHash == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(tokenutils.Hash(t.Secret)), []byte(secretHash)) == 1
}

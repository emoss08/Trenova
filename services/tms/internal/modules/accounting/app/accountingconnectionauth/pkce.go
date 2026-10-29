package accountingconnectionauth

import (
	"errors"
)

var errVerifierMissing = errors.New("the authorization state holds no code verifier")

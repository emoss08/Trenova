package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"io"

	"github.com/emoss08/trenova/shared/hashutils"
)

const keyDerivationLabel = "trenova csrf token key v1"

func Token(sessionID, secret string) string {
	mac := hmac.New(sha256.New, hashutils.DeriveKey(secret, keyDerivationLabel))
	_, _ = io.WriteString(mac, sessionID)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func Verify(token, sessionID, secret string) bool {
	expected := Token(sessionID, secret)
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

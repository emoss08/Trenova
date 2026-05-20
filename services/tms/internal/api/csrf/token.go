package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

func Token(sessionID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(sessionID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func Verify(token, sessionID, secret string) bool {
	expected := Token(sessionID, secret)
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

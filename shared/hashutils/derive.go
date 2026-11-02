package hashutils

import (
	"crypto/hmac"
	"crypto/sha256"
	"io"
)

func DeriveKey(secret, label string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, label)
	return mac.Sum(nil)
}

func WriteParts(w io.Writer, parts ...string) {
	for i, part := range parts {
		if i > 0 {
			_, _ = w.Write([]byte{0})
		}
		_, _ = io.WriteString(w, part)
	}
}

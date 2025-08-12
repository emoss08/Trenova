package grpcutils

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func EncodeToken(name, id string) string {
	raw := fmt.Sprintf("%s|%s", name, id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeToken(tok string) (name, id string, ok bool) {
	b, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

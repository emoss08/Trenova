package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const keyCursorSeparator = ":"

var ErrKeyCursorInvalid = errors.New("key cursor is invalid")

func EncodeKeyCursor(scope, key string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(scope + keyCursorSeparator + key))
}

func DecodeKeyCursor(scope, encoded string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrKeyCursorInvalid, err)
	}

	key, ok := strings.CutPrefix(string(raw), scope+keyCursorSeparator)
	if !ok || key == "" {
		return "", ErrKeyCursorInvalid
	}

	return key, nil
}

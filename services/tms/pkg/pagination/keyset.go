package pagination

import (
	"encoding/base64"
	"errors"
)

var ErrKeysetRequired = errors.New("cursor carries no keyset")

type EdgeCursorProvider interface {
	EdgeCursorAt(index int) (string, bool)
}

func EncodeKeysetCursor(binding CursorBinding, keyset string) (string, error) {
	if keyset == "" {
		return "", ErrKeysetRequired
	}

	raw, err := base64.RawURLEncoding.DecodeString(keyset)
	if err != nil || len(raw) == 0 {
		return "", ErrKeysetRequired
	}

	return sealCursor(cursorKindKeyset, binding, raw), nil
}

func DecodeKeysetCursor(binding CursorBinding, encoded string) (string, error) {
	raw, err := openCursor(cursorKindKeyset, binding, encoded)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

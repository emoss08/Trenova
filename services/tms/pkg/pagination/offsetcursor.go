package pagination

import (
	"strconv"
)

const MaxOffsetCursor = 100_000

func EncodeOffsetCursor(scope string, offset int) string {
	return EncodeKeyCursor(scope, strconv.Itoa(offset))
}

func DecodeOffsetCursor(scope, encoded string) (int, error) {
	if encoded == "" {
		return 0, nil
	}

	key, err := DecodeKeyCursor(scope, encoded)
	if err != nil {
		return 0, err
	}

	offset, err := strconv.Atoi(key)
	if err != nil || offset < 0 || offset > MaxOffsetCursor {
		return 0, ErrKeyCursorInvalid
	}

	return offset, nil
}

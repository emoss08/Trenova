package pagination

import (
	"strconv"
)

const MaxOffsetCursor = 100_000

func EncodeOffsetCursor(binding CursorBinding, offset int) string {
	return sealCursor(cursorKindOffset, binding, strconv.AppendInt(nil, int64(offset), 10))
}

func DecodeOffsetCursor(binding CursorBinding, encoded string) (int, error) {
	if encoded == "" {
		return 0, nil
	}

	raw, err := openCursor(cursorKindOffset, binding, encoded)
	if err != nil {
		return 0, err
	}

	offset, err := strconv.Atoi(string(raw))
	if err != nil || offset < 0 || offset > MaxOffsetCursor {
		return 0, ErrCursorInvalid
	}

	return offset, nil
}

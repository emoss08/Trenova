package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
)

const MaxCursorSkip = 1000

var (
	ErrCursorSortMismatch = errors.New("cursor sort does not match request sort")
	ErrCursorSkipInvalid  = errors.New("cursor skip is out of range")
)

type EdgeCursorProvider interface {
	EdgeCursorAt(index int) (string, bool)
}

func (c *Cursor) IsKeyset() bool {
	return c.Keyset != "" || c.Skip > 0
}

func EncodeKeysetCursor(sort []CursorSortField, keyset string, skip int) (string, error) {
	cursor := Cursor{Sort: sort, Keyset: keyset, Skip: skip}
	if !cursor.IsKeyset() {
		return "", errors.New("keyset cursor needs a keyset or a skip")
	}
	if len(cursor.Values) > 0 || cursor.Skip < 0 {
		return "", ErrCursorSkipInvalid
	}

	bytes, err := sonic.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (c *Cursor) MatchesSort(sort []CursorSortField) bool {
	return slices.Equal(c.Sort, sort)
}

func validateKeysetCursor(cursor *Cursor) error {
	if len(cursor.Values) > 0 {
		return errors.New("keyset cursor carries no values")
	}
	if cursor.Skip < 0 || cursor.Skip > MaxCursorSkip {
		return ErrCursorSkipInvalid
	}

	return nil
}

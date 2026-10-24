package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
)

var (
	ErrCursorSortMismatch = errors.New("cursor sort does not match request sort")
	ErrKeysetRequired     = errors.New("cursor carries no keyset")
)

type EdgeCursorProvider interface {
	EdgeCursorAt(index int) (string, bool)
}

func EncodeKeysetCursor(sort []CursorSortField, keyset string) (string, error) {
	if keyset == "" {
		return "", ErrKeysetRequired
	}

	bytes, err := sonic.Marshal(Cursor{Sort: sort, Keyset: keyset})
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (c *Cursor) MatchesSort(sort []CursorSortField) bool {
	return slices.Equal(c.Sort, sort)
}

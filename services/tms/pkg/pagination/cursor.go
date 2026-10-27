package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrNoEdgeCursor = errors.New("cursor list result has no cursor for its last item")

type CursorSortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Cursor struct {
	Sort   []CursorSortField `json:"sort,omitempty"`
	Keyset string            `json:"keyset"`
}

type CursorEntity interface {
	GetID() pulid.ID
	GetCreatedAt() int64
}

func DecodeCursor(encoded string) (Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor Cursor
	if err = sonic.Unmarshal(bytes, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
	}
	if cursor.Keyset == "" {
		return Cursor{}, ErrKeysetRequired
	}

	return cursor, nil
}

type CursorInfo struct {
	Limit             int
	After             string
	Cursor            Cursor
	IncludeTotalCount bool
}

func NewCursorInfo(first int, after string) (CursorInfo, error) {
	info := CursorInfo{
		Limit:             ClampLimit(first),
		After:             after,
		IncludeTotalCount: true,
	}
	if after == "" {
		return info, nil
	}

	cursor, err := DecodeCursor(after)
	if err != nil {
		return CursorInfo{}, err
	}
	info.Cursor = cursor

	return info, nil
}

type CursorListResult[T any] struct {
	Items           []T
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
	EdgeCursors     []string
	Fuzzy           bool
}

func (r *CursorListResult[T]) EdgeCursorAt(index int) (string, bool) {
	if index < 0 || index >= len(r.EdgeCursors) {
		return "", false
	}

	return r.EdgeCursors[index], true
}

func (r *CursorListResult[T]) NextCursor() (string, error) {
	if !r.HasNextPage || len(r.Items) == 0 {
		return "", nil
	}

	cursor, ok := r.EdgeCursorAt(len(r.Items) - 1)
	if !ok {
		return "", ErrNoEdgeCursor
	}

	return cursor, nil
}

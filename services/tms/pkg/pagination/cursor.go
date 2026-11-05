package pagination

import (
	"errors"

	"github.com/emoss08/trenova/shared/pulid"
)

var ErrNoEdgeCursor = errors.New("cursor list result has no cursor for its last item")

type CursorSortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type CursorEntity interface {
	GetID() pulid.ID
	GetCreatedAt() int64
}

type CursorInfo struct {
	Limit             int
	After             string
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

	if _, ok := cursorBytes(after); !ok {
		return CursorInfo{}, ErrCursorInvalid
	}

	return info, nil
}

type CursorListResult[T any] struct {
	Items           []T
	HasNextPage     bool
	TotalCount      *int
	TotalCountExact bool
	EdgeCursors     EdgeCursors
	Fuzzy           bool
}

type EdgeCursors []string

func (c EdgeCursors) EdgeCursorAt(index int) (string, bool) {
	if index < 0 || index >= len(c) {
		return "", false
	}

	return c[index], true
}

func NextCursor[T any](r *CursorListResult[T]) (string, error) {
	if !r.HasNextPage || len(r.Items) == 0 {
		return "", nil
	}

	cursor, ok := r.EdgeCursors.EdgeCursorAt(len(r.Items) - 1)
	if !ok {
		return "", ErrNoEdgeCursor
	}

	return cursor, nil
}

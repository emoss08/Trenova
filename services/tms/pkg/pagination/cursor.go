package pagination

import (
	"encoding/base64"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
)

type Cursor struct {
	CreatedAt int64    `json:"createdAt"`
	ID        pulid.ID `json:"id"`
}

func EncodeCursor(cursor Cursor) (string, error) {
	if cursor.ID.IsNil() {
		return "", fmt.Errorf("cursor id is required")
	}
	if _, err := pulid.MustParse(cursor.ID.String()); err != nil {
		return "", fmt.Errorf("cursor id is invalid: %w", err)
	}

	bytes, err := sonic.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
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
	if cursor.ID.IsNil() {
		return Cursor{}, fmt.Errorf("cursor id is required")
	}
	if _, err = pulid.MustParse(cursor.ID.String()); err != nil {
		return Cursor{}, fmt.Errorf("cursor id is invalid: %w", err)
	}

	return cursor, nil
}

type CursorInfo struct {
	Limit  int
	After  string
	Cursor Cursor
}

func NewCursorInfo(first int, after string) (CursorInfo, error) {
	info := CursorInfo{
		Limit: ClampLimit(first),
		After: after,
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
	Items       []T
	HasNextPage bool
}

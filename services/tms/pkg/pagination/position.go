package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrPositionIDRequired = errors.New("position cursor id is required")

type PositionCursor struct {
	At int64    `json:"at"`
	ID pulid.ID `json:"id"`
}

func EncodePositionCursor(cursor PositionCursor) (string, error) {
	if cursor.ID.IsNil() {
		return "", ErrPositionIDRequired
	}

	bytes, err := sonic.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal position cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func DecodePositionCursor(encoded string) (PositionCursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return PositionCursor{}, fmt.Errorf("decode position cursor: %w", err)
	}

	var cursor PositionCursor
	if err = sonic.Unmarshal(bytes, &cursor); err != nil {
		return PositionCursor{}, fmt.Errorf("unmarshal position cursor: %w", err)
	}
	if cursor.ID.IsNil() {
		return PositionCursor{}, ErrPositionIDRequired
	}

	return cursor, nil
}

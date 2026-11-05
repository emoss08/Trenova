package pagination

import (
	"encoding/binary"
	"errors"

	"github.com/emoss08/trenova/shared/pulid"
)

var ErrPositionIDRequired = errors.New("position cursor id is required")

type PositionCursor struct {
	At int64
	ID pulid.ID
}

func EncodePositionCursor(binding CursorBinding, cursor PositionCursor) (string, error) {
	if cursor.ID.IsNil() {
		return "", ErrPositionIDRequired
	}

	payload := binary.AppendVarint(nil, cursor.At)
	payload = append(payload, cursor.ID...)

	return sealCursor(cursorKindPosition, binding, payload), nil
}

func DecodePositionCursor(binding CursorBinding, encoded string) (PositionCursor, error) {
	raw, err := openCursor(cursorKindPosition, binding, encoded)
	if err != nil {
		return PositionCursor{}, err
	}

	at, read := binary.Varint(raw)
	if read <= 0 {
		return PositionCursor{}, ErrCursorInvalid
	}

	id := pulid.ID(raw[read:])
	if id.IsNil() {
		return PositionCursor{}, ErrCursorInvalid
	}

	return PositionCursor{At: at, ID: id}, nil
}

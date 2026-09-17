package jsonutils

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
)

type RawJSON []byte

func (r RawJSON) Value() (driver.Value, error) {
	if len(bytes.TrimSpace(r)) == 0 {
		return nil, nil
	}
	if sonic.Valid(r) {
		return string(r), nil
	}
	encoded, err := sonic.Marshal(string(r))
	if err != nil {
		return nil, fmt.Errorf("encode raw JSON: %w", err)
	}
	return string(encoded), nil
}

func (r *RawJSON) Scan(src any) error {
	var data []byte
	switch typed := src.(type) {
	case nil:
		*r = nil
		return nil
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("scan raw JSON: unsupported type %T", src)
	}
	*r = unwrapEncodedJSON(bytes.TrimSpace(data))
	return nil
}

func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return slices.Clone(r), nil
}

func (r *RawJSON) UnmarshalJSON(data []byte) error {
	*r = slices.Clone(data)
	return nil
}

func (r RawJSON) String() string {
	return string(r)
}

func unwrapEncodedJSON(data []byte) RawJSON {
	if len(data) == 0 || data[0] != '"' {
		return slices.Clone(data)
	}
	var inner string
	if err := sonic.Unmarshal(data, &inner); err != nil {
		return slices.Clone(data)
	}
	trimmed := bytes.TrimSpace([]byte(inner))
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') || !sonic.Valid(trimmed) {
		return slices.Clone(data)
	}
	return trimmed
}

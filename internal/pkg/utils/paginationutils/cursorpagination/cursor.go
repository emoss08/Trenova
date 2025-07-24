/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package cursorpagination

import (
	"encoding/base64"

	"github.com/bytedance/sonic"
	"github.com/rotisserie/eris"
)

// EncodeCursor creates an encoded cursor from a model's primary key values
func EncodeCursor(pk PrimaryKey) (string, error) {
	cursor := Cursor{
		Values: make(map[string]any),
	}
	for i, field := range pk.Fields {
		if i < len(pk.Values) {
			cursor.Values[field] = pk.Values[i]
		}
	}
	bytes, err := sonic.Marshal(cursor)
	if err != nil {
		return "", eris.Wrap(err, "marshal cursor")
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// DecodeCursor decodes a cursor string into cursor values
func DecodeCursor(encoded string) (*Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, eris.Wrap(err, "decode base64")
	}
	var cursor Cursor
	if err = sonic.Unmarshal(bytes, &cursor); err != nil {
		return nil, eris.Wrap(err, "unmarshal cursor")
	}
	return &cursor, nil
}

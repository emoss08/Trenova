package jsonutils

import (
	"fmt"

	"github.com/bytedance/sonic"
)

// Convert copies a loosely typed value, such as a map decoded from a request
// or a tool call, into a typed destination by way of JSON. It is the one
// honest way to turn map[string]any into a struct: the struct's own tags
// decide the field names, and a value that does not fit is refused rather
// than silently dropped.
func Convert(src, dst any) error {
	encoded, err := sonic.Marshal(src)
	if err != nil {
		return fmt.Errorf("encode value: %w", err)
	}

	if err = sonic.Unmarshal(encoded, dst); err != nil {
		return fmt.Errorf("decode value: %w", err)
	}

	return nil
}

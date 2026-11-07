package scimproto

import (
	"errors"
	"strings"
)

var errNotBoolean = errors.New("scim: value is not a boolean")

type Bool bool

func BoolOf(value bool) *Bool {
	b := Bool(value)
	return &b
}

func (b Bool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}

	return []byte("false"), nil
}

func (b *Bool) UnmarshalJSON(data []byte) error {
	parsed, err := parseBool(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	*b = Bool(parsed)

	return nil
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.Trim(raw, `"`)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errNotBoolean
	}
}

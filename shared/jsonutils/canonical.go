package jsonutils

import (
	"encoding/json" //nolint:depguard // json.Number is the number type sonic decodes into
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/shopspring/decimal"
)

var canonicalAPI = sonic.Config{SortMapKeys: true, UseNumber: true}.Froze()

// CanonicalValue turns any JSON-shaped value into one form whatever produced
// it: maps of string keys, lists, strings, booleans, nil, and every number as
// its exact decimal. A value read back from jsonb, decoded by sonic or built
// in Go therefore encodes to the same bytes.
func CanonicalValue(value any) (any, error) {
	if typed, ok := value.(map[string]any); ok {
		if typed == nil {
			return nil, nil //nolint:nilnil // a nil map is JSON null, which is a value
		}

		return canonicalMap(typed)
	}

	return canonicalRoundTrip(value)
}

// CanonicalMarshal encodes a value in canonical form: keys sorted, numbers as
// exact decimals, no insignificant whitespace.
func CanonicalMarshal(value any) ([]byte, error) {
	canonical, err := CanonicalValue(value)
	if err != nil {
		return nil, err
	}

	return canonicalAPI.Marshal(canonical)
}

// CanonicalDigest is the lowercase hex SHA-256 of a value's canonical form.
func CanonicalDigest(value any) (string, error) {
	encoded, err := CanonicalMarshal(value)
	if err != nil {
		return "", err
	}

	return hashutils.SHA256BytesHex(encoded), nil
}

func canonicalRoundTrip(value any) (any, error) {
	raw, err := canonicalAPI.Marshal(value)
	if err != nil {
		return nil, err
	}
	if string(raw) == "null" {
		return nil, nil //nolint:nilnil // JSON null is a value
	}

	var decoded any
	if err = canonicalAPI.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	return canonicalElement(decoded)
}

func canonicalMap(in map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for key, value := range in {
		canonical, err := canonicalElement(value)
		if err != nil {
			return nil, err
		}
		out[key] = canonical
	}

	return out, nil
}

func canonicalElement(value any) (any, error) {
	switch typed := value.(type) {
	case nil, string, bool:
		return typed, nil
	case json.Number:
		return canonicalNumber(typed.String())
	case float64:
		return json.Number(decimal.NewFromFloat(typed).String()), nil
	case float32:
		return json.Number(decimal.NewFromFloat32(typed).String()), nil
	case int:
		return json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int32:
		return json.Number(strconv.FormatInt(int64(typed), 10)), nil
	case int64:
		return json.Number(strconv.FormatInt(typed, 10)), nil
	case uint64:
		return json.Number(strconv.FormatUint(typed, 10)), nil
	case map[string]any:
		return canonicalMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			canonical, err := canonicalElement(item)
			if err != nil {
				return nil, err
			}
			out[i] = canonical
		}
		return out, nil
	default:
		return canonicalRoundTrip(typed)
	}
}

func canonicalNumber(raw string) (json.Number, error) {
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return "", fmt.Errorf("canonicalize number %q: %w", raw, err)
	}

	return json.Number(d.String()), nil
}

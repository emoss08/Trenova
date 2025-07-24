/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package jsonutils

import (
	"github.com/bytedance/sonic"
	"github.com/rotisserie/eris"
)

// JSONConverter provides an interface for objects that can convert themselves to JSON
type JSONConverter interface {
	ToJSONString() (string, error)
	ToJSON() (map[string]any, error)
}

// ToJSONString converts any struct to a JSON string
func ToJSONString(v any) (string, error) {
	jsonBytes, err := sonic.Marshal(v)
	if err != nil {
		return "", eris.Wrap(err, "failed to marshal to JSON string")
	}
	return string(jsonBytes), nil
}

// ToJSON converts any struct to a map[string]any
func ToJSON(v any) (map[string]any, error) {
	jsonStr, err := ToJSONString(v)
	if err != nil {
		return nil, err
	}

	var jsonMap map[string]any
	if err = sonic.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal to map")
	}
	return jsonMap, nil
}

// MustToJSONString converts any struct to a JSON string, panics on error
// Use this only when you're absolutely certain the conversion won't fail
func MustToJSONString(v any) string {
	str, err := ToJSONString(v)
	if err != nil {
		panic(err)
	}
	return str
}

// MustToJSON converts any struct to a map[string]any, panics on error
// Use this only when you're absolutely certain the conversion won't fail
func MustToJSON(v any) map[string]any {
	m, err := ToJSON(v)
	if err != nil {
		panic(err)
	}
	return m
}

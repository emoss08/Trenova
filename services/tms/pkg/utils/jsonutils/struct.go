package jsonutils

import (
	"fmt"

	"github.com/bytedance/sonic"
)

type JSONConverter interface {
	ToJSONString() (string, error)
	ToJSON() (map[string]any, error)
}

func ToJSONString(v any) (string, error) {
	jsonBytes, err := sonic.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON string: %w", err)
	}
	return string(jsonBytes), nil
}

func ToJSON(v any) (map[string]any, error) {
	jsonStr, err := ToJSONString(v)
	if err != nil {
		return nil, err
	}

	var jsonMap map[string]any
	if err = sonic.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}
	return jsonMap, nil
}

func MustToJSONString(v any) string {
	str, err := ToJSONString(v)
	if err != nil {
		panic(err)
	}
	return str
}

func MustToJSON(v any) map[string]any {
	m, err := ToJSON(v)
	if err != nil {
		panic(err)
	}
	return m
}

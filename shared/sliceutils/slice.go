package sliceutils

import "strings"

func FirstOrDefault[T any](slice []T, fallback T) T {
	if len(slice) > 0 {
		return slice[0]
	}
	return fallback
}

func StringSliceValue(v any) []string {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]string); ok {
			return append([]string{}, direct...)
		}
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := StringValue(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func StringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

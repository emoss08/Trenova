package sliceutils

import "strings"

func FirstOrDefault[T any](slice []T, fallback T) T {
	if len(slice) > 0 {
		return slice[0]
	}
	return fallback
}

func Difference[T comparable](items, exclude []T) []T {
	if len(items) == 0 {
		return nil
	}

	excluded := make(map[T]struct{}, len(exclude))
	for _, item := range exclude {
		excluded[item] = struct{}{}
	}

	result := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := excluded[item]; !ok {
			result = append(result, item)
		}
	}

	return result
}

func Dedupe[T comparable](items []T) []T {
	if len(items) == 0 {
		return []T{}
	}
	seen := make(map[T]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
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

func StringPtrValue(v any) *string {
	value := StringValue(v)
	if value == "" {
		return nil
	}
	return &value
}

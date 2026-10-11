package sim

import (
	"net/http"
	"strings"
)

type routeEntry[T any] struct {
	Method string
	Path   string
	Value  T
}

type routeTable[T any] struct {
	byKey map[string]T
}

func newRouteTable[T any](entries []routeEntry[T]) routeTable[T] {
	byKey := make(map[string]T, len(entries))
	for _, entry := range entries {
		byKey[routeKey(entry.Method, entry.Path)] = entry.Value
	}
	return routeTable[T]{byKey: byKey}
}

func (t routeTable[T]) lookup(key string) (T, bool) {
	value, ok := t.byKey[key]
	return value, ok
}

func routeKey(method, path string) string {
	cleanPath := strings.TrimSpace(path)
	var builder strings.Builder
	builder.Grow(len(method) + len(cleanPath) + 1)
	builder.WriteString(strings.ToUpper(strings.TrimSpace(method)))
	builder.WriteByte(' ')
	for idx, segment := range strings.Split(strings.Trim(cleanPath, "/"), "/") {
		if idx > 0 || cleanPath != "" {
			builder.WriteByte('/')
		}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			builder.WriteString("{}")
			continue
		}
		builder.WriteString(segment)
	}
	return builder.String()
}

func patternRouteKey(pattern, fallbackMethod string) string {
	clean := strings.TrimSpace(pattern)
	method, path, found := strings.Cut(clean, " ")
	if !found || strings.HasPrefix(method, "/") {
		return routeKey(fallbackMethod, clean)
	}
	return routeKey(method, strings.TrimSpace(path))
}

func requestRouteKey(request *http.Request) string {
	if request.Pattern != "" {
		return patternRouteKey(request.Pattern, request.Method)
	}
	return routeKey(request.Method, request.URL.Path)
}

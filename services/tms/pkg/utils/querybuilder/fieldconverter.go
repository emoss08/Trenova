package querybuilder

import (
	"maps"
	"reflect"
	"strings"
	"sync"

	"github.com/emoss08/trenova/pkg/utils"
)

// Global cache for field extraction to avoid repeated reflection
var (
	fieldCache   = make(map[reflect.Type]map[string]string)
	fieldCacheMu sync.RWMutex
)

func ExtractFieldsFromStruct(v any) map[string]string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return make(map[string]string)
	}

	// Check cache first
	fieldCacheMu.RLock()
	if cached, exists := fieldCache[t]; exists {
		fieldCacheMu.RUnlock()
		result := make(map[string]string, len(cached))
		maps.Copy(result, cached)
		return result
	}
	fieldCacheMu.RUnlock()

	fieldMap := make(map[string]string)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			continue
		}

		dbFieldName := extractDBFieldName(field.Tag.Get("bun"), jsonName)
		fieldMap[jsonName] = dbFieldName
	}

	fieldCacheMu.Lock()
	fieldCache[t] = fieldMap
	fieldCacheMu.Unlock()

	result := make(map[string]string, len(fieldMap))
	maps.Copy(result, fieldMap)
	return result
}

func extractDBFieldName(bunTag, jsonName string) string {
	if bunTag == "" || bunTag == "-" {
		return utils.ConvertCamelToSnake(jsonName)
	}

	parts := strings.Split(bunTag, ",")
	if len(parts) == 0 {
		return utils.ConvertCamelToSnake(jsonName)
	}

	if !strings.Contains(parts[0], ":") {
		return parts[0]
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, ":") &&
			part != "pk" &&
			part != "notnull" &&
			part != "unique" &&
			part != "default" {
			return part
		}
	}

	return utils.ConvertCamelToSnake(jsonName)
}

func WarmFieldCache(entities ...any) {
	for i := range len(entities) {
		_ = ExtractFieldsFromStruct(entities[i])
	}
}

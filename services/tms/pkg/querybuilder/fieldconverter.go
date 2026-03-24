package querybuilder

import (
	"maps"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

type StaticFieldMapper interface {
	GetStaticFieldMap() map[string]string
}

// ExtractFieldsFromStruct returns a mapping of JSON field names to database column names
// for the given entity. The returned map is always a fresh copy safe for mutation.
//
// If the entity implements [StaticFieldMapper] (via generated code), the pre-computed map
// is copied and returned — this skips both reflection and caching.
func ExtractFieldsFromStruct(v any) map[string]string {
	if v == nil {
		return make(map[string]string)
	}

	if mapper, ok := v.(StaticFieldMapper); ok {
		src := mapper.GetStaticFieldMap()
		dst := make(map[string]string, len(src))
		maps.Copy(dst, src)
		return dst
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return make(map[string]string)
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return make(map[string]string)
	}

	return getOrComputeFieldMap(t, func() map[string]string {
		return extractFieldsFromType(t)
	})
}

func extractFieldsFromType(t reflect.Type) map[string]string {
	fieldMap := make(map[string]string)

	for field := range t.Fields() {

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

	return fieldMap
}

func extractDBFieldName(bunTag, jsonName string) string {
	if bunTag == "" || bunTag == "-" {
		return stringutils.ConvertCamelToSnake(jsonName)
	}

	parts := strings.Split(bunTag, ",")
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

	return stringutils.ConvertCamelToSnake(jsonName)
}

func WarmFieldCache(entities ...any) {
	for _, entity := range entities {
		_ = ExtractFieldsFromStruct(entity)
	}
}

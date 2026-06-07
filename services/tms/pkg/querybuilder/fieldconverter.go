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

func ExtractNonNullableFieldsFromStruct(v any) map[string]bool {
	return extractFieldMetadataFromStruct(v, fieldIsNonNullable)
}

func ExtractIntegerFieldsFromStruct(v any) map[string]bool {
	return extractFieldMetadataFromStruct(v, fieldIsInteger)
}

func extractFieldMetadataFromStruct(
	v any,
	include func(reflect.StructField) bool,
) map[string]bool {
	result := make(map[string]bool)
	if v == nil {
		return result
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return result
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return result
	}

	for field := range t.Fields() {
		if !field.IsExported() || !include(field) {
			continue
		}

		jsonName := jsonFieldName(field.Tag.Get("json"))
		if jsonName == "" {
			continue
		}
		result[jsonName] = true
	}

	return result
}

func extractFieldsFromType(t reflect.Type) map[string]string {
	fieldMap := make(map[string]string)

	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}

		jsonName := jsonFieldName(field.Tag.Get("json"))
		if jsonName == "" {
			continue
		}

		dbFieldName := extractDBFieldName(field.Tag.Get("bun"), jsonName)
		fieldMap[jsonName] = dbFieldName
	}

	return fieldMap
}

func jsonFieldName(jsonTag string) string {
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}

	return strings.Split(jsonTag, ",")[0]
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

func fieldIsNonNullable(field reflect.StructField) bool {
	if field.Type.Kind() == reflect.Pointer {
		return false
	}

	bunTag := field.Tag.Get("bun")
	if bunTag == "" || bunTag == "-" || strings.Contains(bunTag, "nullzero") {
		return false
	}

	return bunTagHasOption(bunTag, "notnull") || bunTagHasOption(bunTag, "pk")
}

func fieldIsInteger(field reflect.StructField) bool {
	fieldType := field.Type
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func bunTagHasOption(bunTag, option string) bool {
	for _, part := range strings.Split(bunTag, ",") {
		if strings.TrimSpace(part) == option {
			return true
		}
	}

	return false
}

func WarmFieldCache(entities ...any) {
	for _, entity := range entities {
		_ = ExtractFieldsFromStruct(entity)
	}
}

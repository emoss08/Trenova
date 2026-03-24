package bunmarshal

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bytedance/sonic"
)

// UnmarshalMap decodes a snake_case Bun-shaped map into a struct that uses JSON tags.
func UnmarshalMap(input map[string]any, target any) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}

	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Pointer || targetType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	translated := translateMap(input, targetType.Elem())
	payload, err := sonic.Marshal(translated)
	if err != nil {
		return fmt.Errorf("marshal translated payload: %w", err)
	}

	if err := sonic.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("unmarshal translated payload: %w", err)
	}

	return nil
}

func translateMap(input map[string]any, targetType reflect.Type) map[string]any {
	fields := fieldMappings(targetType)
	output := make(map[string]any, len(input))

	for key, value := range input {
		if strings.HasPrefix(key, "_") {
			continue
		}

		if name, ok := fields[key]; ok {
			output[name] = value
			continue
		}

		output[key] = value
	}

	return output
}

func fieldMappings(targetType reflect.Type) map[string]string {
	fields := make(map[string]string)

	for idx := range targetType.NumField() {
		field := targetType.Field(idx)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if field.Anonymous {
			embeddedType := field.Type
			if embeddedType.Kind() == reflect.Pointer {
				embeddedType = embeddedType.Elem()
			}
			if embeddedType.Kind() == reflect.Struct {
				for key, value := range fieldMappings(embeddedType) {
					fields[key] = value
				}
			}
			continue
		}

		bunName := parseTagName(field.Tag.Get("bun"))
		jsonName := parseTagName(field.Tag.Get("json"))

		if bunName == "" || bunName == "-" || jsonName == "" || jsonName == "-" {
			continue
		}

		fields[bunName] = jsonName
	}

	return fields
}

func parseTagName(tag string) string {
	if tag == "" {
		return ""
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return ""
	}

	return name
}

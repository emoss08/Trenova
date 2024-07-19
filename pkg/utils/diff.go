package utils

import "reflect"

// FieldChange represents a change in a single field
type FieldChange struct {
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

// DiffStructs compares two structs and returns a map of the changes
func DiffStructs(original, updated interface{}) map[string]FieldChange {
	changes := make(map[string]FieldChange)

	originalValue := reflect.ValueOf(original)
	updatedValue := reflect.ValueOf(updated)

	// If either value is a pointer, get the value it points to
	if originalValue.Kind() == reflect.Ptr {
		originalValue = originalValue.Elem()
	}
	if updatedValue.Kind() == reflect.Ptr {
		updatedValue = updatedValue.Elem()
	}

	// Ensure both are structs
	if originalValue.Kind() != reflect.Struct || updatedValue.Kind() != reflect.Struct {
		return changes
	}

	for i := 0; i < originalValue.NumField(); i++ {
		fieldName := originalValue.Type().Field(i).Name
		originalField := originalValue.Field(i)
		updatedField := updatedValue.Field(i)

		// Skip unexported fields
		if !originalField.CanInterface() {
			continue
		}

		// Compare the fields
		if !reflect.DeepEqual(originalField.Interface(), updatedField.Interface()) {
			changes[fieldName] = FieldChange{
				From: originalField.Interface(),
				To:   updatedField.Interface(),
			}
		}
	}

	return changes
}

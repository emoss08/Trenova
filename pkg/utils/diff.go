package utils

import (
	"encoding/json"
	"reflect"
)

// FieldChange represents a change in a single field
type FieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

// DiffStructs compares two structs and returns a map of the changes
func StructDiff(original, updated any) map[string]FieldChange {
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

// JSONDiff compares two structs and returns a map of the changes
func JSONDiff(before, after any) (map[string]FieldChange, error) {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return nil, err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return nil, err
	}

	var beforeMap, afterMap map[string]any
	if err = json.Unmarshal(beforeJSON, &beforeMap); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(afterJSON, &afterMap); err != nil {
		return nil, err
	}

	diff := make(map[string]FieldChange)
	for key, afterValue := range afterMap {
		beforeValue, exists := beforeMap[key]
		if !exists {
			// This is a new field
			diff[key] = FieldChange{
				To: afterValue,
			}
		} else if !reflect.DeepEqual(beforeValue, afterValue) {
			// This is an updated field
			diff[key] = FieldChange{
				From: beforeValue,
				To:   afterValue,
			}
		}
	}

	// Check for deleted fields
	for key, beforeValue := range beforeMap {
		if _, exists := afterMap[key]; !exists {
			diff[key] = FieldChange{
				From: beforeValue,
			}
		}
	}

	return diff, nil
}

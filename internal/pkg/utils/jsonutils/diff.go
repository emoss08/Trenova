// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package jsonutils

import (
	"fmt"
	"reflect"
	"time"

	"slices"

	"github.com/bytedance/sonic"
	"github.com/rotisserie/eris"
)

// ChangeType represents the type of change that occurred
type ChangeType string

const (
	ChangeTypeCreated ChangeType = "created"
	ChangeTypeUpdated ChangeType = "updated"
	ChangeTypeDeleted ChangeType = "deleted"
)

// FieldType represents the data type of the field
type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeNumber    FieldType = "number"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeArray     FieldType = "array"
	FieldTypeObject    FieldType = "object"
	FieldTypeDateTime  FieldType = "datetime"
	FieldTypeNull      FieldType = "null"
	FieldTypeUndefined FieldType = "undefined"
)

// FieldChange represents a change in a single field with additional metadata
type FieldChange struct {
	From      any        `json:"from"`
	To        any        `json:"to"`
	Type      ChangeType `json:"type"`
	FieldType FieldType  `json:"fieldType"`
	Path      string     `json:"path"`
}

// DiffOptions provides configuration options for the diff operation
type DiffOptions struct {
	IgnoreFields    []string // Fields to ignore during comparison
	CustomComparors map[string]Comparator
	MaxDepth        int  // Maximum depth for nested object comparison
	IgnoreCase      bool // Whether to ignore case in string comparisons
}

// Comparator is a function type that defines custom comparison logic
type Comparator func(before, after any) (bool, error)

// DefaultOptions returns the default DiffOptions
func DefaultOptions() *DiffOptions {
	return &DiffOptions{
		IgnoreFields: []string{},
		CustomComparors: map[string]Comparator{
			"time": compareTime,
		},
		MaxDepth:   10,
		IgnoreCase: false,
	}
}

// JSONDiff compares two structs and returns a map of the changes
func JSONDiff(before, after any, opts *DiffOptions) (map[string]FieldChange, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	beforeJSON, err := sonic.Marshal(before)
	if err != nil {
		return nil, eris.Wrap(err, "failed to marshal before value")
	}
	afterJSON, err := sonic.Marshal(after)
	if err != nil {
		return nil, eris.Wrap(err, "failed to marshal after value")
	}

	var beforeMap, afterMap map[string]any
	if err = sonic.Unmarshal(beforeJSON, &beforeMap); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal before value")
	}
	if err = sonic.Unmarshal(afterJSON, &afterMap); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal after value")
	}

	diff := make(map[string]FieldChange)
	if err = compareObjects(beforeMap, afterMap, "", diff, opts, 0); err != nil {
		return nil, eris.Wrap(err, "failed to compare objects")
	}

	return diff, nil
}

func compareObjects(
	before, after map[string]any,
	path string,
	diff map[string]FieldChange,
	opts *DiffOptions,
	depth int,
) error {
	if depth > opts.MaxDepth {
		return eris.New("max depth exceeded")
	}

	// Check for created and updated fields
	if err := compareCreatedAndUpdated(before, after, path, diff, opts, depth); err != nil {
		return err
	}

	// Check for deleted fields
	compareDeleted(before, after, path, diff, opts)

	return nil
}

// compareCreatedAndUpdated handles fields that were created or updated in the after object
func compareCreatedAndUpdated(
	before, after map[string]any,
	path string,
	diff map[string]FieldChange,
	opts *DiffOptions,
	depth int,
) error {
	for key, afterValue := range after {
		if shouldIgnoreField(key, opts.IgnoreFields) {
			continue
		}

		currentPath := joinPath(path, key)
		beforeValue, exists := before[key]

		if !exists {
			recordCreatedField(currentPath, afterValue, diff)
			continue
		}

		if err := compareExistingField(beforeValue, afterValue, key, currentPath, diff, opts, depth); err != nil {
			return err
		}
	}
	return nil
}

// recordCreatedField adds a created field to the diff
func recordCreatedField(path string, value any, diff map[string]FieldChange) {
	diff[path] = FieldChange{
		To:        value,
		Type:      ChangeTypeCreated,
		FieldType: determineFieldType(value),
		Path:      path,
	}
}

// compareExistingField compares a field that exists in both before and after objects
func compareExistingField(
	beforeValue, afterValue any,
	key, path string,
	diff map[string]FieldChange,
	opts *DiffOptions,
	depth int,
) error {
	// Check if there's a custom comparator for this field
	if comparator, ok := opts.CustomComparors[key]; ok {
		return handleCustomComparison(comparator, beforeValue, afterValue, key, path, diff)
	}

	// Handle nested objects
	if isObject(beforeValue) && isObject(afterValue) {
		beforeMap, afterMap := toMap(beforeValue), toMap(afterValue)
		return compareObjects(beforeMap, afterMap, path, diff, opts, depth+1)
	}

	// Handle arrays
	if isArray(beforeValue) && isArray(afterValue) {
		if !reflect.DeepEqual(beforeValue, afterValue) {
			diff[path] = createFieldChange(beforeValue, afterValue, path)
		}
		return nil
	}

	// Handle primitive values
	if !compareValues(beforeValue, afterValue, opts.IgnoreCase) {
		diff[path] = createFieldChange(beforeValue, afterValue, path)
	}

	return nil
}

// handleCustomComparison applies a custom comparator and records changes if needed
func handleCustomComparison(
	comparator Comparator,
	beforeValue, afterValue any,
	key, path string,
	diff map[string]FieldChange,
) error {
	equal, err := comparator(beforeValue, afterValue)
	if err != nil {
		return eris.Wrapf(err, "custom comparison failed for field %s", key)
	}
	if !equal {
		diff[path] = createFieldChange(beforeValue, afterValue, path)
	}
	return nil
}

// compareDeleted identifies and records fields that were deleted
func compareDeleted(
	before, after map[string]any,
	path string,
	diff map[string]FieldChange,
	opts *DiffOptions,
) {
	for key, beforeValue := range before {
		if shouldIgnoreField(key, opts.IgnoreFields) {
			continue
		}

		currentPath := joinPath(path, key)
		if _, exists := after[key]; !exists {
			diff[currentPath] = FieldChange{
				From:      beforeValue,
				Type:      ChangeTypeDeleted,
				FieldType: determineFieldType(beforeValue),
				Path:      currentPath,
			}
		}
	}
}

func createFieldChange(before, after any, path string) FieldChange {
	return FieldChange{
		From:      before,
		To:        after,
		Type:      ChangeTypeUpdated,
		FieldType: determineFieldType(after),
		Path:      path,
	}
}

func determineFieldType(value any) FieldType {
	if value == nil {
		return FieldTypeNull
	}

	switch v := value.(type) {
	case string:
		return FieldTypeString
	case float64, float32, int, int64, int32:
		return FieldTypeNumber
	case bool:
		return FieldTypeBoolean
	case []any:
		return FieldTypeArray
	case map[string]any:
		return FieldTypeObject
	case time.Time:
		return FieldTypeDateTime
	default:
		rt := reflect.TypeOf(v)
		if rt == nil {
			return FieldTypeUndefined
		}
		//nolint:exhaustive // we want to handle all the cases
		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			return FieldTypeArray
		case reflect.Map, reflect.Struct:
			return FieldTypeObject
		case reflect.Bool:
			return FieldTypeBoolean
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
			return FieldTypeNumber
		case reflect.String:
			return FieldTypeString
		default:
			return FieldTypeString
		}
	}
}

func shouldIgnoreField(field string, ignoreFields []string) bool {
	return slices.Contains(ignoreFields, field)
}

func compareValues(before, after any, ignoreCase bool) bool {
	if ignoreCase {
		beforeStr, beforeOK := before.(string)
		afterStr, afterOK := after.(string)
		if beforeOK && afterOK {
			return compareStringsIgnoreCase(beforeStr, afterStr)
		}
	}
	return reflect.DeepEqual(before, after)
}

func compareStringsIgnoreCase(a, b string) bool {
	return a == b
}

func compareTime(before, after any) (bool, error) {
	beforeTime, ok1 := before.(time.Time)
	afterTime, ok2 := after.(time.Time)

	if !ok1 || !ok2 {
		return false, eris.New("invalid time comparison")
	}

	return beforeTime.Equal(afterTime), nil
}

func isObject(v any) bool {
	switch v.(type) {
	case map[string]any:
		return true
	default:
		return false
	}
}

func isArray(v any) bool {
	switch v.(type) {
	case []any:
		return true
	default:
		return false
	}
}

func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", base, key)
}

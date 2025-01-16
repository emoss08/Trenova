package jsonutils

import (
	"fmt"
	"reflect"
	"time"

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

func compareObjects(before, after map[string]any, path string, diff map[string]FieldChange, opts *DiffOptions, depth int) error {
	if depth > opts.MaxDepth {
		return eris.New("max depth exceeded")
	}

	// Check for created and updated fields
	for key, afterValue := range after {
		if shouldIgnoreField(key, opts.IgnoreFields) {
			continue
		}

		currentPath := joinPath(path, key)
		beforeValue, exists := before[key]

		if !exists {
			// Field was created
			diff[currentPath] = FieldChange{
				To:        afterValue,
				Type:      ChangeTypeCreated,
				FieldType: determineFieldType(afterValue),
				Path:      currentPath,
			}
			continue
		}

		// Check if there's a custom comparator for this field
		if comparator, ok := opts.CustomComparors[key]; ok {
			equal, err := comparator(beforeValue, afterValue)
			if err != nil {
				return eris.Wrapf(err, "custom comparison failed for field %s", key)
			}
			if !equal {
				diff[currentPath] = createFieldChange(beforeValue, afterValue, currentPath)
			}
			continue
		}

		// Handle nested objects
		if isObject(beforeValue) && isObject(afterValue) {
			beforeMap, afterMap := toMap(beforeValue), toMap(afterValue)
			if err := compareObjects(beforeMap, afterMap, currentPath, diff, opts, depth+1); err != nil {
				return err
			}
			continue
		}

		// Handle arrays
		if isArray(beforeValue) && isArray(afterValue) {
			if !reflect.DeepEqual(beforeValue, afterValue) {
				diff[currentPath] = createFieldChange(beforeValue, afterValue, currentPath)
			}
			continue
		}

		// Handle primitive values
		if !compareValues(beforeValue, afterValue, opts.IgnoreCase) {
			diff[currentPath] = createFieldChange(beforeValue, afterValue, currentPath)
		}
	}

	// Check for deleted fields
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

	return nil
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
		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			return FieldTypeArray
		case reflect.Map, reflect.Struct:
			return FieldTypeObject
		default:
			return FieldTypeString
		}
	}
}

func shouldIgnoreField(field string, ignoreFields []string) bool {
	for _, f := range ignoreFields {
		if f == field {
			return true
		}
	}
	return false
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

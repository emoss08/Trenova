package jsonutils

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/bytedance/sonic"
)

var (
	ErrMaxDepthExceeded      = errors.New("max depth exceeded")
	ErrInvalidTimeComparison = errors.New("invalid time comparison")
)

type ChangeType string

const (
	ChangeTypeCreated ChangeType = "created"
	ChangeTypeUpdated ChangeType = "updated"
	ChangeTypeDeleted ChangeType = "deleted"
)

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

type FieldChange struct {
	From      any        `json:"from"`
	To        any        `json:"to"`
	Type      ChangeType `json:"type"`
	FieldType FieldType  `json:"fieldType"`
	Path      string     `json:"path"`
}

type DiffOptions struct {
	IgnoreFields    []string // Fields to ignore during comparison
	CustomComparors map[string]Comparator
	MaxDepth        int  // Maximum depth for nested object comparison
	IgnoreCase      bool // Whether to ignore case in string comparisons
}

type Comparator func(before, after any) (bool, error)

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

func JSONDiff(before, after any, opts *DiffOptions) (map[string]FieldChange, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	beforeJSON, err := sonic.Marshal(before)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal before value: %w", err)
	}
	afterJSON, err := sonic.Marshal(after)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal after value: %w", err)
	}

	var beforeMap, afterMap map[string]any
	if err = sonic.Unmarshal(beforeJSON, &beforeMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal before value: %w", err)
	}
	if err = sonic.Unmarshal(afterJSON, &afterMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal after value: %w", err)
	}

	diff := make(map[string]FieldChange)
	if err = compareObjects(beforeMap, afterMap, "", diff, opts, 0); err != nil {
		return nil, fmt.Errorf("failed to compare objects: %w", err)
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
		return ErrMaxDepthExceeded
	}

	if err := compareCreatedAndUpdated(before, after, path, diff, opts, depth); err != nil {
		return err
	}
	compareDeleted(before, after, path, diff, opts)

	return nil
}

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

func recordCreatedField(path string, value any, diff map[string]FieldChange) {
	diff[path] = FieldChange{
		To:        value,
		Type:      ChangeTypeCreated,
		FieldType: determineFieldType(value),
		Path:      path,
	}
}

func compareExistingField(
	beforeValue, afterValue any,
	key, path string,
	diff map[string]FieldChange,
	opts *DiffOptions,
	depth int,
) error {
	if comparator, ok := opts.CustomComparors[key]; ok {
		return handleCustomComparison(comparator, beforeValue, afterValue, key, path, diff)
	}

	if isObject(beforeValue) && isObject(afterValue) {
		beforeMap, afterMap := toMap(beforeValue), toMap(afterValue)
		return compareObjects(beforeMap, afterMap, path, diff, opts, depth+1)
	}

	if isArray(beforeValue) && isArray(afterValue) {
		if !reflect.DeepEqual(beforeValue, afterValue) {
			diff[path] = createFieldChange(beforeValue, afterValue, path)
		}
		return nil
	}

	if !compareValues(beforeValue, afterValue, opts.IgnoreCase) {
		diff[path] = createFieldChange(beforeValue, afterValue, path)
	}

	return nil
}

func handleCustomComparison(
	comparator Comparator,
	beforeValue, afterValue any,
	key, path string,
	diff map[string]FieldChange,
) error {
	equal, err := comparator(beforeValue, afterValue)
	if err != nil {
		return fmt.Errorf("custom comparison failed for field %s: %w", key, err)
	}
	if !equal {
		diff[path] = createFieldChange(beforeValue, afterValue, path)
	}
	return nil
}

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
		switch rt.Kind() { //nolint:exhaustive // TODO: add more types
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
		return false, ErrInvalidTimeComparison
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

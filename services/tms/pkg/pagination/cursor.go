package pagination

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
)

type CursorSortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type CursorValueColumn struct {
	SQLExpression string
	Alias         string
}

type Cursor struct {
	CreatedAt int64             `json:"createdAt,omitempty"`
	Sort      []CursorSortField `json:"sort,omitempty"`
	Values    []any             `json:"values,omitempty"`
	ID        pulid.ID          `json:"id"`
}

type CursorEntity interface {
	GetID() pulid.ID
	GetCreatedAt() int64
}

type CursorValueCarrier interface {
	CursorValues(count int) []any
}

type CursorValueProvider interface {
	CursorValuesAt(index int) ([]any, bool)
}

type CursorValueSet struct {
	CursorValue0 any `json:"-" bun:"__cursor_value_0,scanonly"`
	CursorValue1 any `json:"-" bun:"__cursor_value_1,scanonly"`
	CursorValue2 any `json:"-" bun:"__cursor_value_2,scanonly"`
	CursorValue3 any `json:"-" bun:"__cursor_value_3,scanonly"`
	CursorValue4 any `json:"-" bun:"__cursor_value_4,scanonly"`
	CursorValue5 any `json:"-" bun:"__cursor_value_5,scanonly"`
	CursorValue6 any `json:"-" bun:"__cursor_value_6,scanonly"`
	CursorValue7 any `json:"-" bun:"__cursor_value_7,scanonly"`
	CursorValue8 any `json:"-" bun:"__cursor_value_8,scanonly"`
	CursorValue9 any `json:"-" bun:"__cursor_value_9,scanonly"`
}

func (s CursorValueSet) CursorValues(count int) []any {
	values := []any{
		s.CursorValue0,
		s.CursorValue1,
		s.CursorValue2,
		s.CursorValue3,
		s.CursorValue4,
		s.CursorValue5,
		s.CursorValue6,
		s.CursorValue7,
		s.CursorValue8,
		s.CursorValue9,
	}

	if count > len(values) {
		count = len(values)
	}
	if count < 0 {
		count = 0
	}

	return append([]any(nil), values[:count]...)
}

func EncodeCursor(cursor Cursor) (string, error) {
	if cursor.ID.IsNil() {
		return "", fmt.Errorf("cursor id is required")
	}
	if _, err := pulid.MustParse(cursor.ID.String()); err != nil {
		return "", fmt.Errorf("cursor id is invalid: %w", err)
	}
	if len(cursor.Sort) > 0 && len(cursor.Sort) != len(cursor.Values) {
		return "", fmt.Errorf("cursor sort values do not match cursor sort shape")
	}

	bytes, err := sonic.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func DecodeCursor(encoded string) (Cursor, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor Cursor
	if err = sonic.Unmarshal(bytes, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("unmarshal cursor: %w", err)
	}
	if cursor.ID.IsNil() {
		return Cursor{}, fmt.Errorf("cursor id is required")
	}
	if _, err = pulid.MustParse(cursor.ID.String()); err != nil {
		return Cursor{}, fmt.Errorf("cursor id is invalid: %w", err)
	}
	if len(cursor.Sort) > 0 && len(cursor.Sort) != len(cursor.Values) {
		return Cursor{}, fmt.Errorf("cursor sort values do not match cursor sort shape")
	}

	return cursor, nil
}

func ValidateCursorSort(cursor Cursor, sort []CursorSortField) error {
	if len(cursor.Sort) != len(sort) {
		return fmt.Errorf("cursor sort does not match request sort")
	}

	for i := range sort {
		if cursor.Sort[i] != sort[i] {
			return fmt.Errorf("cursor sort does not match request sort")
		}
	}

	return nil
}

type CursorInfo struct {
	Limit  int
	After  string
	Cursor Cursor
}

func NewCursorInfo(first int, after string) (CursorInfo, error) {
	info := CursorInfo{
		Limit: ClampLimit(first),
		After: after,
	}
	if after == "" {
		return info, nil
	}

	cursor, err := DecodeCursor(after)
	if err != nil {
		return CursorInfo{}, err
	}
	info.Cursor = cursor

	return info, nil
}

type CursorListResult[T any] struct {
	Items        []T
	HasNextPage  bool
	TotalCount   *int
	CursorSort   []CursorSortField
	CursorValues [][]any
}

func NewCursorListResult[T any](items []T, limit int) *CursorListResult[T] {
	return NewCursorListResultWithTotalCount(items, limit, nil)
}

func NewCursorListResultWithTotalCount[T any](
	items []T,
	limit int,
	totalCount *int,
) *CursorListResult[T] {
	hasNextPage := len(items) > limit
	if hasNextPage {
		items = items[:limit]
	}

	return &CursorListResult[T]{
		Items:       items,
		HasNextPage: hasNextPage,
		TotalCount:  totalCount,
	}
}

func (r *CursorListResult[T]) WithCursorSort(sort []CursorSortField) *CursorListResult[T] {
	r.CursorSort = append([]CursorSortField(nil), sort...)
	return r
}

func (r *CursorListResult[T]) WithCursorValues(values [][]any) error {
	if len(values) < len(r.Items) {
		return fmt.Errorf("cursor values do not match result items")
	}

	r.CursorValues = make([][]any, len(r.Items))
	for i := range r.Items {
		r.CursorValues[i] = append([]any(nil), values[i]...)
	}

	return nil
}

func (r *CursorListResult[T]) CursorValuesAt(index int) ([]any, bool) {
	if index < 0 || index >= len(r.CursorValues) {
		return nil, false
	}

	return r.CursorValues[index], true
}

func EncodeCursorFromEntity[T any](item T) (string, error) {
	id, err := cursorEntityID(item)
	if err != nil {
		return "", err
	}
	createdAt, err := cursorEntityCreatedAt(item)
	if err != nil {
		return "", err
	}

	return EncodeCursor(Cursor{
		CreatedAt: createdAt,
		ID:        id,
	})
}

func EncodeCursorFromEntityWithSort[T any](
	item T,
	sort []CursorSortField,
) (string, error) {
	values, err := ExtractCursorValues(item, sort)
	if err != nil {
		return "", err
	}
	id, err := cursorEntityID(item)
	if err != nil {
		return "", err
	}
	createdAt, err := cursorEntityCreatedAt(item)
	if err != nil {
		return "", err
	}

	return EncodeCursor(Cursor{
		CreatedAt: createdAt,
		Sort:      sort,
		Values:    values,
		ID:        id,
	})
}

func EncodeCursorFromEntityWithValues[T any](
	item T,
	sort []CursorSortField,
	values []any,
) (string, error) {
	if len(sort) == 0 {
		return EncodeCursorFromEntity(item)
	}
	if len(values) != len(sort) {
		return "", fmt.Errorf("cursor sort values do not match cursor sort shape")
	}

	normalizedValues := make([]any, 0, len(values))
	for _, value := range values {
		normalizedValues = append(normalizedValues, normalizeCursorValue(value))
	}

	id, err := cursorIDFromSortValues(sort, normalizedValues)
	if err != nil {
		return "", err
	}
	createdAt := cursorCreatedAtFromSortValues(sort, normalizedValues)
	if createdAt == 0 {
		createdAt, _ = cursorEntityCreatedAt(item)
	}

	return EncodeCursor(Cursor{
		CreatedAt: createdAt,
		Sort:      sort,
		Values:    normalizedValues,
		ID:        id,
	})
}

func ExtractCursorValues[T any](item T, sort []CursorSortField) ([]any, error) {
	values := make([]any, 0, len(sort))
	for _, field := range sort {
		if field.Field == "id" {
			id, err := cursorEntityID(item)
			if err != nil {
				return nil, err
			}
			values = append(values, id.String())
			continue
		}

		value, err := fieldPathValue(reflect.ValueOf(item), field.Field)
		if err != nil {
			return nil, err
		}
		values = append(values, normalizeCursorValue(value))
	}

	return values, nil
}

func cursorIDFromSortValues(sort []CursorSortField, values []any) (pulid.ID, error) {
	for i, field := range sort {
		if field.Field != "id" {
			continue
		}

		switch typed := values[i].(type) {
		case pulid.ID:
			if typed.IsNil() {
				return pulid.Nil, fmt.Errorf("cursor id is required")
			}
			return typed, nil
		case string:
			id, err := pulid.MustParse(typed)
			if err != nil {
				return pulid.Nil, fmt.Errorf("cursor id is invalid: %w", err)
			}
			return id, nil
		default:
			return pulid.Nil, fmt.Errorf("cursor field %q is not a pulid id", "id")
		}
	}

	return pulid.Nil, fmt.Errorf("cursor id is required")
}

func cursorCreatedAtFromSortValues(sort []CursorSortField, values []any) int64 {
	for i, field := range sort {
		if field.Field != "createdAt" {
			continue
		}

		switch typed := values[i].(type) {
		case int64:
			return typed
		case int:
			return int64(typed)
		case int32:
			return int64(typed)
		case float64:
			return int64(typed)
		case string:
			createdAt, err := strconv.ParseInt(typed, 10, 64)
			if err == nil {
				return createdAt
			}
		}
	}

	return 0
}

func cursorEntityID(item any) (pulid.ID, error) {
	if entity, ok := item.(CursorEntity); ok {
		return entity.GetID(), nil
	}

	value, err := fieldPathValue(reflect.ValueOf(item), "id")
	if err != nil {
		return pulid.Nil, err
	}

	switch typed := value.(type) {
	case pulid.ID:
		return typed, nil
	case string:
		id, parseErr := pulid.MustParse(typed)
		if parseErr != nil {
			return pulid.Nil, fmt.Errorf("cursor id is invalid: %w", parseErr)
		}
		return id, nil
	default:
		return pulid.Nil, fmt.Errorf("cursor field %q is not a pulid id", "id")
	}
}

func cursorEntityCreatedAt(item any) (int64, error) {
	if entity, ok := item.(CursorEntity); ok {
		return entity.GetCreatedAt(), nil
	}

	value, err := fieldPathValue(reflect.ValueOf(item), "createdAt")
	if err != nil {
		return 0, err
	}

	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	case string:
		createdAt, parseErr := strconv.ParseInt(typed, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("cursor field %q is invalid: %w", "createdAt", parseErr)
		}
		return createdAt, nil
	default:
		return 0, fmt.Errorf("cursor field %q is not an int64 timestamp", "createdAt")
	}
}

func fieldPathValue(value reflect.Value, path string) (any, error) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}

	parts := splitFieldPath(path)
	for _, part := range parts {
		if value.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cursor field %q is not available", path)
		}

		next, ok := structJSONField(value, part)
		if !ok {
			return nil, fmt.Errorf("cursor field %q is not available", path)
		}
		value = next
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil, nil
			}
			value = value.Elem()
		}
	}

	if !value.IsValid() {
		return nil, nil
	}

	return value.Interface(), nil
}

func splitFieldPath(path string) []string {
	parts := make([]string, 0, 2)
	start := 0
	for i := range path {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	return append(parts, path[start:])
}

func structJSONField(value reflect.Value, jsonName string) (reflect.Value, bool) {
	valueType := value.Type()
	for i := range value.NumField() {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		tagName := field.Tag.Get("json")
		if comma := indexByte(tagName, ','); comma >= 0 {
			tagName = tagName[:comma]
		}
		if tagName == jsonName {
			return value.Field(i), true
		}
	}

	return reflect.Value{}, false
}

func indexByte(value string, target byte) int {
	for i := range value {
		if value[i] == target {
			return i
		}
	}
	return -1
}

func normalizeCursorValue(value any) any {
	switch typed := value.(type) {
	case pulid.ID:
		if typed.IsNil() {
			return nil
		}
		return typed.String()
	default:
		return value
	}
}

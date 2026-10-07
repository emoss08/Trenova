package base

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	memoryDefaultPage = 20
	memoryMaxPage     = 200
	memoryCursorTag   = "mem:"
)

// MemoryField is one column of an in-memory connection: how to read it from a
// row, for filtering and sorting.
type MemoryField[T any] func(row T) any

// MemoryTable describes rows a server has already worked out, such as an
// aggregate, so the data table can search, filter, sort and page them the
// way it does any other connection.
type MemoryTable[T any] struct {
	Fields map[string]MemoryField[T]
	// Search is the text a query is matched against; nil disables search.
	Search func(row T) string
}

// MemoryPage is one page of rows, with the total that matched.
type MemoryPage[T any] struct {
	// Start is the offset of the first row of the page among the matches.
	Start     int
	Rows      []T
	Total     int
	HasNext   bool
	EndCursor *string
}

// Page applies the data table's input to rows: search, then field filters
// (all must hold), then filter groups (each group holds when any of its
// filters does), then sort, then the page after the cursor. A filter or sort
// on a field the table does not describe is refused, as it is for a query.
func (m *MemoryTable[T]) Page(input *gqlmodel.DataTableConnectionInput, rows []T) (*MemoryPage[T], error) {
	if input == nil {
		input = &gqlmodel.DataTableConnectionInput{}
	}

	matched := make([]T, 0, len(rows))
	for _, row := range rows {
		keep, err := m.matches(input, row)
		if err != nil {
			return nil, err
		}
		if keep {
			matched = append(matched, row)
		}
	}

	if err := m.sort(input.Sort, matched); err != nil {
		return nil, err
	}

	offset, err := decodeMemoryCursor(input.After)
	if err != nil {
		return nil, err
	}
	size := memoryDefaultPage
	if input.First != nil && *input.First > 0 {
		size = min(*input.First, memoryMaxPage)
	}

	start := min(offset, len(matched))
	end := min(start+size, len(matched))
	page := &MemoryPage[T]{
		Start:   start,
		Rows:    matched[start:end],
		Total:   len(matched),
		HasNext: end < len(matched),
	}
	if end > start {
		cursor := encodeMemoryCursor(end)
		page.EndCursor = &cursor
	}
	return page, nil
}

func (m *MemoryTable[T]) matches(input *gqlmodel.DataTableConnectionInput, row T) (bool, error) {
	if input.Query != nil && strings.TrimSpace(*input.Query) != "" && m.Search != nil {
		if !strings.Contains(strings.ToLower(m.Search(row)), strings.ToLower(strings.TrimSpace(*input.Query))) {
			return false, nil
		}
	}

	for _, filter := range input.FieldFilters {
		ok, err := m.holds(filter, row)
		if err != nil || !ok {
			return false, err
		}
	}

	for _, group := range input.FilterGroups {
		if group == nil || len(group.Filters) == 0 {
			continue
		}
		any := false
		for _, filter := range group.Filters {
			ok, err := m.holds(filter, row)
			if err != nil {
				return false, err
			}
			any = any || ok
		}
		if !any {
			return false, nil
		}
	}

	return true, nil
}

func (m *MemoryTable[T]) holds(filter *gqlmodel.FieldFilterInput, row T) (bool, error) {
	if filter == nil {
		return true, nil
	}
	read, ok := m.Fields[filter.Field]
	if !ok {
		return false, errortypes.NewValidationError("fieldFilters", errortypes.ErrInvalid,
			"This table cannot be filtered by {0}", filter.Field)
	}
	return compareOperator(filter.Operator, read(row), filter.Value)
}

func (m *MemoryTable[T]) sort(fields []*gqlmodel.SortFieldInput, rows []T) error {
	for _, field := range fields {
		if _, ok := m.Fields[field.Field]; !ok {
			return errortypes.NewValidationError("sort", errortypes.ErrInvalid,
				"This table cannot be sorted by {0}", field.Field)
		}
	}
	if len(fields) == 0 {
		return nil
	}

	slices.SortStableFunc(rows, func(a, b T) int {
		for _, field := range fields {
			read := m.Fields[field.Field]
			order := compareValues(read(a), read(b))
			if strings.EqualFold(field.Direction, "desc") {
				order = -order
			}
			if order != 0 {
				return order
			}
		}
		return 0
	})
	return nil
}

func compareOperator(operator string, value, want any) (bool, error) {
	switch operator {
	case "eq":
		return compareValues(value, want) == 0, nil
	case "ne":
		return compareValues(value, want) != 0, nil
	case "gt":
		return compareValues(value, want) > 0, nil
	case "gte":
		return compareValues(value, want) >= 0, nil
	case "lt":
		return compareValues(value, want) < 0, nil
	case "lte":
		return compareValues(value, want) <= 0, nil
	case "contains":
		return strings.Contains(strings.ToLower(fmt.Sprint(value)), strings.ToLower(fmt.Sprint(want))), nil
	case "isnull":
		return value == nil, nil
	case "isnotnull":
		return value != nil, nil
	default:
		return false, errortypes.NewValidationError("fieldFilters", errortypes.ErrInvalid,
			"This table does not filter with {0}", operator)
	}
}

// compareValues orders numbers as numbers and anything else as text.
func compareValues(a, b any) int {
	left, leftNumber := asNumber(a)
	right, rightNumber := asNumber(b)
	if leftNumber && rightNumber {
		return cmp.Compare(left, right)
	}
	return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
}

func asNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case string:
		parsed, err := strconv.ParseFloat(number, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func encodeMemoryCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(memoryCursorTag + strconv.Itoa(offset)))
}

func decodeMemoryCursor(after *string) (int, error) {
	if after == nil || *after == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(*after)
	if err != nil || !strings.HasPrefix(string(raw), memoryCursorTag) {
		return 0, errortypes.NewValidationError("after", errortypes.ErrInvalid, "Invalid cursor")
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(string(raw), memoryCursorTag))
	if err != nil || offset < 0 {
		return 0, errortypes.NewValidationError("after", errortypes.ErrInvalid, "Invalid cursor")
	}
	return offset, nil
}

// CursorAt is the cursor of the page's row at index, which a reader resumes
// after.
func (p *MemoryPage[T]) CursorAt(index int) string {
	return encodeMemoryCursor(p.Start + index + 1)
}

package memtable

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	MaxQueryLength = 200
	MaxFilters     = 50
	MaxSortFields  = 5
)

type Kind uint8

const (
	KindText Kind = iota
	KindEnum
	KindSet
	KindBoolean
	KindNumber
)

type Field[T any] struct {
	Name       string
	Kind       Kind
	Values     []string
	Filterable bool
	Sortable   bool
	Text       func(*T) string
	Set        func(*T) []string
	Bool       func(*T) bool
	Number     func(*T) (float64, bool)
	Compare    func(a, b *T) int
}

type Config[T any] struct {
	CursorScope string
	Search      func(*T) string
	Order       func(a, b *T) int
	Fields      []Field[T]
}

type Table[T any] struct {
	scope  string
	search func(*T) string
	order  func(a, b *T) int
	fields map[string]*Field[T]
}

type Request struct {
	Query             string
	FieldFilters      []domaintypes.FieldFilter
	FilterGroups      []domaintypes.FilterGroup
	Sort              []domaintypes.SortField
	First             int
	After             string
	IncludeTotalCount bool
}

type Page[T any] struct {
	Items       []*T
	Cursors     []string
	HasNextPage bool
	TotalCount  *int
}

type predicate[T any] func(*T) bool

type comparator[T any] func(a, b *T) int

func New[T any](cfg Config[T]) *Table[T] {
	fields := make(map[string]*Field[T], len(cfg.Fields))
	for idx := range cfg.Fields {
		field := cfg.Fields[idx]
		field.Values = normalizedValues(field.Values)
		fields[field.Name] = &field
	}

	return &Table[T]{
		scope:  cfg.CursorScope,
		search: cfg.Search,
		order:  cfg.Order,
		fields: fields,
	}
}

func (t *Table[T]) References(req *Request, name string) bool {
	if req == nil {
		return false
	}
	for idx := range req.FieldFilters {
		if req.FieldFilters[idx].Field == name {
			return true
		}
	}
	for idx := range req.FilterGroups {
		for inner := range req.FilterGroups[idx].Filters {
			if req.FilterGroups[idx].Filters[inner].Field == name {
				return true
			}
		}
	}
	for idx := range req.Sort {
		if req.Sort[idx].Field == name {
			return true
		}
	}

	return false
}

func (t *Table[T]) List(items []T, req *Request) (*Page[T], error) {
	if req == nil {
		req = &Request{}
	}

	plan, err := t.compile(req)
	if err != nil {
		return nil, err
	}

	offset, err := pagination.DecodeOffsetCursor(t.scope, req.After)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalidFormat,
			"Cursor is invalid",
		)
	}

	matched := make([]*T, 0, len(items))
	for idx := range items {
		item := &items[idx]
		if plan.matches(item) {
			matched = append(matched, item)
		}
	}

	slices.SortStableFunc(matched, plan.compare)

	limit := pageSize(req.First)
	page := &Page[T]{}
	if req.IncludeTotalCount {
		total := len(matched)
		page.TotalCount = &total
	}
	if offset >= len(matched) {
		page.Items = []*T{}
		page.Cursors = []string{}

		return page, nil
	}

	end := min(offset+limit, len(matched))
	page.Items = matched[offset:end]
	page.HasNextPage = end < len(matched)
	page.Cursors = make([]string, 0, len(page.Items))
	for idx := range page.Items {
		page.Cursors = append(
			page.Cursors,
			pagination.EncodeOffsetCursor(t.scope, offset+idx+1),
		)
	}

	return page, nil
}

func pageSize(first int) int {
	if first <= 0 {
		return pagination.DefaultLimit
	}

	return pagination.ClampLimit(first)
}

type plan[T any] struct {
	query   string
	search  func(*T) string
	filters []predicate[T]
	groups  [][]predicate[T]
	sorts   []comparator[T]
	order   func(a, b *T) int
}

func (p *plan[T]) matches(item *T) bool {
	if p.query != "" && p.search != nil &&
		!strings.Contains(strings.ToLower(p.search(item)), p.query) {
		return false
	}
	for _, filter := range p.filters {
		if !filter(item) {
			return false
		}
	}
	for _, group := range p.groups {
		if !slices.ContainsFunc(group, func(filter predicate[T]) bool { return filter(item) }) {
			return false
		}
	}

	return true
}

func (p *plan[T]) compare(a, b *T) int {
	for _, sort := range p.sorts {
		if result := sort(a, b); result != 0 {
			return result
		}
	}
	if p.order != nil {
		return p.order(a, b)
	}

	return 0
}

func (t *Table[T]) compile(req *Request) (*plan[T], error) {
	multiErr := errortypes.NewMultiError()

	query := strings.TrimSpace(req.Query)
	if utf8.RuneCountInString(query) > MaxQueryLength {
		multiErr.Add("query", errortypes.ErrInvalid, "Search must be at most 200 characters")
	}

	filterCount := len(req.FieldFilters)
	for idx := range req.FilterGroups {
		filterCount += len(req.FilterGroups[idx].Filters)
	}
	if filterCount > MaxFilters {
		multiErr.Add("fieldFilters", errortypes.ErrInvalid, "At most 50 filters can be applied")
	}
	if len(req.Sort) > MaxSortFields {
		multiErr.Add("sort", errortypes.ErrInvalid, "At most 5 sort fields can be applied")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	out := &plan[T]{
		query:   strings.ToLower(query),
		search:  t.search,
		filters: make([]predicate[T], 0, len(req.FieldFilters)),
		groups:  make([][]predicate[T], 0, len(req.FilterGroups)),
		sorts:   make([]comparator[T], 0, len(req.Sort)),
		order:   t.order,
	}

	for idx := range req.FieldFilters {
		filter, ok := t.compileFilter(
			multiErr,
			fmt.Sprintf("fieldFilters[%d]", idx),
			&req.FieldFilters[idx],
		)
		if ok {
			out.filters = append(out.filters, filter)
		}
	}

	for groupIdx := range req.FilterGroups {
		group := req.FilterGroups[groupIdx].Filters
		if len(group) == 0 {
			continue
		}

		compiled := make([]predicate[T], 0, len(group))
		for idx := range group {
			filter, ok := t.compileFilter(
				multiErr,
				fmt.Sprintf("filterGroups[%d].filters[%d]", groupIdx, idx),
				&group[idx],
			)
			if ok {
				compiled = append(compiled, filter)
			}
		}
		out.groups = append(out.groups, compiled)
	}

	for idx := range req.Sort {
		sort, ok := t.compileSort(multiErr, fmt.Sprintf("sort[%d]", idx), &req.Sort[idx])
		if ok {
			out.sorts = append(out.sorts, sort)
		}
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return out, nil
}

func (t *Table[T]) compileSort(
	multiErr *errortypes.MultiError,
	path string,
	sort *domaintypes.SortField,
) (comparator[T], bool) {
	field, ok := t.fields[sort.Field]
	if !ok || !field.Sortable {
		multiErr.Add(path+".field", errortypes.ErrInvalid, "{0} cannot be sorted", sort.Field)

		return nil, false
	}
	if !sort.Direction.IsValid() {
		multiErr.Add(
			path+".direction",
			errortypes.ErrInvalid,
			"Sort direction must be asc or desc",
		)

		return nil, false
	}

	base := field.comparator()
	if base == nil {
		multiErr.Add(path+".field", errortypes.ErrInvalid, "{0} cannot be sorted", sort.Field)

		return nil, false
	}

	desc := sort.Direction == dbtype.SortDirectionDesc
	missing := field.missing()

	return func(a, b *T) int {
		if missing != nil {
			missingA, missingB := missing(a), missing(b)
			switch {
			case missingA && missingB:
				return 0
			case missingA:
				return 1
			case missingB:
				return -1
			}
		}
		if desc {
			return base(b, a)
		}

		return base(a, b)
	}, true
}

func (f *Field[T]) missing() func(*T) bool {
	if f.Kind != KindNumber || f.Number == nil || f.Compare != nil {
		return nil
	}

	return func(item *T) bool {
		_, ok := f.Number(item)
		return !ok
	}
}

func (f *Field[T]) comparator() comparator[T] {
	if f.Compare != nil {
		return f.Compare
	}

	switch f.Kind {
	case KindText, KindEnum:
		if f.Text == nil {
			return nil
		}

		return func(a, b *T) int {
			return cmp.Compare(strings.ToLower(f.Text(a)), strings.ToLower(f.Text(b)))
		}
	case KindBoolean:
		if f.Bool == nil {
			return nil
		}

		return func(a, b *T) int { return compareBool(f.Bool(a), f.Bool(b)) }
	case KindNumber:
		if f.Number == nil {
			return nil
		}

		return func(a, b *T) int {
			left, _ := f.Number(a)
			right, _ := f.Number(b)

			return cmp.Compare(left, right)
		}
	default:
		return nil
	}
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	default:
		return -1
	}
}

func (t *Table[T]) compileFilter(
	multiErr *errortypes.MultiError,
	path string,
	filter *domaintypes.FieldFilter,
) (predicate[T], bool) {
	field, ok := t.fields[filter.Field]
	if !ok || !field.Filterable {
		multiErr.Add(path+".field", errortypes.ErrInvalid, "{0} cannot be filtered", filter.Field)

		return nil, false
	}

	var (
		compiled predicate[T]
		err      error
	)
	switch field.Kind {
	case KindText:
		compiled, err = textPredicate(field, filter)
	case KindEnum:
		compiled, err = enumPredicate(field, filter)
	case KindSet:
		compiled, err = setPredicate(field, filter)
	case KindBoolean:
		compiled, err = boolPredicate(field, filter)
	case KindNumber:
		compiled, err = numberPredicate(field, filter)
	default:
		err = errUnsupportedOperator
	}
	if err != nil {
		addFilterError(multiErr, path, filter.Field, err)

		return nil, false
	}

	return compiled, true
}

func addFilterError(multiErr *errortypes.MultiError, path, field string, err error) {
	switch {
	case errors.Is(err, errUnsupportedOperator):
		multiErr.Add(
			path+".operator",
			errortypes.ErrInvalid,
			"{0} cannot be filtered with this operator",
			field,
		)
	case errors.Is(err, errUnknownValue):
		multiErr.Add(path+".value", errortypes.ErrInvalid, "{0} has no such value", field)
	default:
		multiErr.Add(path+".value", errortypes.ErrInvalid, "{0} filter value is invalid", field)
	}
}

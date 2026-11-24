package querytool

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const ParamWhere = "__where"

type Where struct {
	Filters []domaintypes.FieldFilter `json:"filters,omitempty"`
	Groups  []domaintypes.FilterGroup `json:"groups,omitempty"`
	Sort    []domaintypes.SortField   `json:"sort,omitempty"`
	Extras  map[string]any            `json:"extras,omitempty"`
}

type Extra struct {
	Name        string
	Kind        string
	Description string
	Operators   []string
	Values      []string
}

type WhereReader interface {
	WhereFields() *domaintypes.FieldConfiguration
}

type WhereExtender interface {
	WhereExtras() []Extra
}

func WhereOf(params map[string]any) *Where {
	where, _ := params[ParamWhere].(*Where)

	return where
}

func WithWhere(params map[string]any, where *Where) map[string]any {
	out := make(map[string]any, len(params)+1)
	for key, value := range params {
		out[key] = value
	}
	out[ParamWhere] = where

	return out
}

func (w *Where) ApplyFilters(filters []domaintypes.FieldFilter) []domaintypes.FieldFilter {
	if w == nil {
		return filters
	}

	return append(filters, w.Filters...)
}

func (w *Where) ApplySort(sorting []domaintypes.SortField) []domaintypes.SortField {
	if w == nil || len(w.Sort) == 0 {
		return sorting
	}

	return w.Sort
}

func (w *Where) FilterGroups() []domaintypes.FilterGroup {
	if w == nil {
		return nil
	}

	return w.Groups
}

package iftarepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func when(include bool, p keel.Pred) keel.Pred {
	if !include {
		return nil
	}

	return p
}

func withDefaultSort(
	filter *pagination.QueryOptions,
	sorts ...domaintypes.SortField,
) *pagination.QueryOptions {
	out := pagination.QueryOptions{}
	if filter != nil {
		out = *filter
	}
	if len(out.Sort) == 0 {
		out.Sort = sorts
	}

	return &out
}

func desc(field string) domaintypes.SortField {
	return domaintypes.SortField{Field: field, Direction: dbtype.SortDirectionDesc}
}

func asc(field string) domaintypes.SortField {
	return domaintypes.SortField{Field: field, Direction: dbtype.SortDirectionAsc}
}

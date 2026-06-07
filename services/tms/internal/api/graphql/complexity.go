package graphql

import (
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func complexityRoot() generated.ComplexityRoot {
	var root generated.ComplexityRoot
	root.Query.Trailers = func(
		childComplexity int,
		first *int,
		_ *string,
		_ *string,
		_ []*gqlmodel.FieldFilterInput,
		_ []*gqlmodel.FilterGroupInput,
		_ []*gqlmodel.SortFieldInput,
		_ *domaintypes.EquipmentStatus,
		_ *bool,
		_ *bool,
	) int {
		return listComplexity(childComplexity, first)
	}
	root.Query.Tractors = func(
		childComplexity int,
		first *int,
		_ *string,
		_ *string,
		_ []*gqlmodel.FieldFilterInput,
		_ []*gqlmodel.FilterGroupInput,
		_ []*gqlmodel.SortFieldInput,
		_ *domaintypes.EquipmentStatus,
		_ *bool,
		_ *bool,
		_ *bool,
	) int {
		return listComplexity(childComplexity, first)
	}

	return root
}

func listComplexity(childComplexity int, first *int) int {
	limit := pagination.DefaultLimit
	if first != nil {
		limit = *first
	}

	return countComplexity(childComplexity, pagination.ClampLimit(limit))
}

func countComplexity(childComplexity, count int) int {
	return count * childComplexity
}

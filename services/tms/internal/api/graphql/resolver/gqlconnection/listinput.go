package gqlconnection

import (
	"context"
	"slices"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GqlListOptions struct {
	TenantInfo   pagination.TenantInfo
	Limit        int
	Offset       int
	Query        string
	FieldFilters []*gqlmodel.FieldFilterInput
	FilterGroups []*gqlmodel.FilterGroupInput
	Sort         []*gqlmodel.SortFieldInput
}

func QueryOptionsFromGraphQL(opts *GqlListOptions) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: opts.TenantInfo,
		Pagination: pagination.Info{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		Query:        opts.Query,
		FieldFilters: FieldFiltersFromGraphQL(opts.FieldFilters),
		FilterGroups: FilterGroupsFromGraphQL(opts.FilterGroups),
		Sort:         SortFieldsFromGraphQL(opts.Sort),
	}
}

type GqlCursorPageInput struct {
	First *int
	After *string
}

type gqlEntityCursorPage struct {
	Cursor pagination.CursorInfo
}

type GqlDataTableConnection struct {
	Filter *pagination.QueryOptions
	Cursor pagination.CursorInfo
}

func DataTableConnectionFromGraphQL(
	ctx context.Context,
	input *gqlmodel.DataTableConnectionInput,
	tenantInfo pagination.TenantInfo,
) (GqlDataTableConnection, error) {
	if input == nil {
		input = &gqlmodel.DataTableConnectionInput{}
	}

	page, err := EntityCursorPageFromGraphQL(ctx, GqlCursorPageInput{
		First: input.First,
		After: input.After,
	})
	if err != nil {
		return GqlDataTableConnection{}, err
	}

	queryValue := ""
	if input.Query != nil {
		queryValue = *input.Query
	}

	filter := QueryOptionsFromGraphQL(&GqlListOptions{
		TenantInfo:   tenantInfo,
		Limit:        page.Cursor.Limit,
		Query:        queryValue,
		FieldFilters: input.FieldFilters,
		FilterGroups: input.FilterGroups,
		Sort:         input.Sort,
	})

	return GqlDataTableConnection{
		Filter: filter,
		Cursor: page.Cursor,
	}, nil
}

func MemtableRequestFromGraphQL(
	ctx context.Context,
	input *gqlmodel.DataTableConnectionInput,
) memtable.Request {
	if input == nil {
		input = &gqlmodel.DataTableConnectionInput{}
	}

	return memtable.Request{
		Query:             gqlscalar.StringValue(input.Query),
		FieldFilters:      FieldFiltersFromGraphQL(input.FieldFilters),
		FilterGroups:      FilterGroupsFromGraphQL(input.FilterGroups),
		Sort:              SortFieldsFromGraphQL(input.Sort),
		First:             gqlscalar.IntValue(input.First),
		After:             gqlscalar.StringValue(input.After),
		IncludeTotalCount: ConnectionTotalRequested(ctx),
	}
}

func EntityCursorPageFromGraphQL(
	ctx context.Context,
	input GqlCursorPageInput,
) (gqlEntityCursorPage, error) {
	limit := pagination.DefaultLimit
	if input.First != nil {
		limit = pagination.ClampLimit(*input.First)
	}

	afterValue := ""
	if input.After != nil {
		afterValue = *input.After
	}
	cursor, err := pagination.NewCursorInfo(limit, afterValue)
	if err != nil {
		return gqlEntityCursorPage{}, errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalidFormat,
			"Cursor is invalid",
		)
	}
	cursor.IncludeTotalCount = ConnectionTotalRequested(ctx)

	return gqlEntityCursorPage{Cursor: cursor}, nil
}

const (
	ConnectionTotalCountField      = "totalCount"
	ConnectionTotalCountExactField = "totalCountExact"
)

func ConnectionTotalRequested(ctx context.Context) bool {
	return ConnectionFieldRequested(
		ctx,
		ConnectionTotalCountField,
		ConnectionTotalCountExactField,
	)
}

func ConnectionFieldRequested(ctx context.Context, names ...string) bool {
	if ctx == nil {
		return true
	}
	if fc := graphql.GetFieldContext(ctx); fc == nil || fc.Field.Field == nil {
		return true
	}

	for _, field := range graphql.CollectFieldsCtx(ctx, nil) {
		if slices.Contains(names, field.Name) {
			return true
		}
	}

	return false
}

func FieldFiltersFromGraphQL(inputs []*gqlmodel.FieldFilterInput) []domaintypes.FieldFilter {
	if len(inputs) == 0 {
		return nil
	}

	filters := make([]domaintypes.FieldFilter, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		filters = append(filters, domaintypes.FieldFilter{
			Field:    input.Field,
			Operator: dbtype.Operator(input.Operator),
			Value: pagination.NormalizeFilterValue(
				input.Value,
				input.Operator,
			),
		})
	}

	return filters
}

func FilterGroupsFromGraphQL(inputs []*gqlmodel.FilterGroupInput) []domaintypes.FilterGroup {
	if len(inputs) == 0 {
		return nil
	}

	groups := make([]domaintypes.FilterGroup, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		groups = append(groups, domaintypes.FilterGroup{
			Filters: FieldFiltersFromGraphQL(input.Filters),
		})
	}

	return groups
}

func SortFieldsFromGraphQL(inputs []*gqlmodel.SortFieldInput) []domaintypes.SortField {
	if len(inputs) == 0 {
		return nil
	}

	sorts := make([]domaintypes.SortField, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		sorts = append(sorts, domaintypes.SortField{
			Field:     input.Field,
			Direction: dbtype.SortDirection(input.Direction),
		})
	}

	return sorts
}

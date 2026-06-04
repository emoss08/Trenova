package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func parseIDs(values []string) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id, err := pulid.MustParse(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func optionalID(value *string) (pulid.ID, error) {
	if value == nil || *value == "" {
		return pulid.Nil, nil
	}
	return pulid.MustParse(*value)
}

func idPtr(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func intPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

type gqlListOptions struct {
	TenantInfo   pagination.TenantInfo
	Limit        int
	Offset       int
	Query        string
	FieldFilters []*gqlmodel.FieldFilterInput
	FilterGroups []*gqlmodel.FilterGroupInput
	Sort         []*gqlmodel.SortFieldInput
}

func queryOptionsFromGraphQL(opts gqlListOptions) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: opts.TenantInfo,
		Pagination: pagination.Info{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		Query:        opts.Query,
		FieldFilters: fieldFiltersFromGraphQL(opts.FieldFilters),
		FilterGroups: filterGroupsFromGraphQL(opts.FilterGroups),
		Sort:         sortFieldsFromGraphQL(opts.Sort),
	}
}

func fieldFiltersFromGraphQL(inputs []*gqlmodel.FieldFilterInput) []domaintypes.FieldFilter {
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

func filterGroupsFromGraphQL(inputs []*gqlmodel.FilterGroupInput) []domaintypes.FilterGroup {
	if len(inputs) == 0 {
		return nil
	}

	groups := make([]domaintypes.FilterGroup, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		groups = append(groups, domaintypes.FilterGroup{
			Filters: fieldFiltersFromGraphQL(input.Filters),
		})
	}

	return groups
}

func sortFieldsFromGraphQL(inputs []*gqlmodel.SortFieldInput) []domaintypes.SortField {
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

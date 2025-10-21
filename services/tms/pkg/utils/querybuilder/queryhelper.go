package querybuilder

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

func ApplyFilters[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	filter *pagination.QueryOptions,
	entity T,
) *bun.SelectQuery {
	fieldConfig := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		WithRelationshipFields().
		Build()

	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)

	qb.ApplyTenantFilters(filter.TenantOpts)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if filter.Query != "" {
		searchFields := ExtractSearchFields(fieldConfig)
		qb.ApplyTextSearch(filter.Query, searchFields)
	}

	if len(filter.Sort) > 0 {
		qb.ApplySort(filter.Sort)
	}

	return qb.GetQuery()
}

func ApplyFiltersWithConfig[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	fieldConfig *pagination.FieldConfiguration,
	filter *pagination.QueryOptions,
	entity T,
) *bun.SelectQuery {
	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)
	qb.ApplyTenantFilters(filter.TenantOpts)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if filter.Query != "" {
		searchFields := ExtractSearchFields(fieldConfig)
		qb.ApplyTextSearch(filter.Query, searchFields)
	}

	if len(filter.Sort) > 0 {
		qb.ApplySort(filter.Sort)
	}

	return qb.GetQuery()
}

func ExtractSearchFields(fieldConfig *pagination.FieldConfiguration) []string {
	searchFields := make([]string, 0)
	for field, filterable := range fieldConfig.FilterableFields {
		if filterable && !fieldConfig.EnumMap[field] {
			searchFields = append(searchFields, field)
		}
	}
	return searchFields
}

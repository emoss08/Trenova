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
	fieldConfig := GetFieldConfiguration(entity)

	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)

	qb.ApplyTenantFilters(filter.TenantInfo)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if len(filter.FilterGroups) > 0 {
		qb.ApplyFilterGroups(filter.FilterGroups)
	}

	if len(filter.GeoFilters) > 0 {
		qb.ApplyGeoFilters(filter.GeoFilters)
	}

	if len(filter.AggregateFilters) > 0 {
		qb.ApplyAggregateFilters(filter.AggregateFilters)
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
	fieldConfig *domaintypes.FieldConfiguration,
	filter *pagination.QueryOptions,
	entity T,
) *bun.SelectQuery {
	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.WithTraversalSupport(true)
	qb.ApplyTenantFilters(filter.TenantInfo)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if len(filter.FilterGroups) > 0 {
		qb.ApplyFilterGroups(filter.FilterGroups)
	}

	if len(filter.GeoFilters) > 0 {
		qb.ApplyGeoFilters(filter.GeoFilters)
	}

	if len(filter.AggregateFilters) > 0 {
		qb.ApplyAggregateFilters(filter.AggregateFilters)
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

func GetDistinctValues[T domaintypes.PostgresSearchable](
	query *bun.SelectQuery,
	tableAlias string,
	field string,
	filter *pagination.QueryOptions,
	entity T,
) *bun.SelectQuery {
	fieldConfig := GetFieldConfiguration(entity)

	qb := NewWithPostgresSearch(query, tableAlias, fieldConfig, entity)
	qb.ApplyTenantFilters(filter.TenantInfo)

	return qb.GetDistinctValues(field).GetQuery()
}

func ExtractSearchFields(fieldConfig *domaintypes.FieldConfiguration) []string {
	searchFields := make([]string, 0, len(fieldConfig.FilterableFields))
	for field, filterable := range fieldConfig.FilterableFields {
		if filterable && !fieldConfig.EnumMap[field] {
			searchFields = append(searchFields, field)
		}
	}
	return searchFields
}

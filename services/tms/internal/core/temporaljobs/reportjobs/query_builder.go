package reportjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/pkg/utils/reportuils"
	"github.com/uptrace/bun"
)

type QueryBuilder struct {
	resourceInfo *ResourceInfo
}

func NewQueryBuilder(resourceType string) (*QueryBuilder, error) {
	resourceInfo, err := GetResourceInfo(resourceType)
	if err != nil {
		return nil, fmt.Errorf("unsupported resource type: %w", err)
	}

	return &QueryBuilder{
		resourceInfo: &resourceInfo,
	}, nil
}

func (qb *QueryBuilder) BuildBaseQuery(
	db *bun.DB,
	organizationID,
	businessUnitID pulid.ID,
) *bun.SelectQuery {
	return db.NewSelect().
		TableExpr(fmt.Sprintf("%s AS %s", qb.resourceInfo.TableName, qb.resourceInfo.Alias)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(fmt.Sprintf("%s.organization_id = ?", qb.resourceInfo.Alias), organizationID).
				Where(fmt.Sprintf("%s.business_unit_id = ?", qb.resourceInfo.Alias), businessUnitID)
		})
}

func (qb *QueryBuilder) ApplyFilters(
	query *bun.SelectQuery,
	queryOptions pagination.QueryOptions,
) *bun.SelectQuery {
	fieldConfig := querybuilder.NewFieldConfigBuilder(qb.resourceInfo.Entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		Build()

	qbuilder := querybuilder.NewWithPostgresSearch(
		query,
		qb.resourceInfo.Alias,
		fieldConfig,
		qb.resourceInfo.Entity,
	)
	qbuilder.WithTraversalSupport(true)

	if len(queryOptions.FieldFilters) > 0 {
		qbuilder.ApplyFilters(queryOptions.FieldFilters)
	}

	if len(queryOptions.Sort) > 0 {
		qbuilder.ApplySort(queryOptions.Sort)
	}

	if queryOptions.Query != "" {
		searchConfig := qb.resourceInfo.Entity.GetPostgresSearchConfig()
		if len(searchConfig.SearchableFields) > 0 {
			searchFields := make([]string, len(searchConfig.SearchableFields))
			for i, field := range searchConfig.SearchableFields {
				searchFields[i] = field.Name
			}
			qbuilder.ApplyTextSearch(queryOptions.Query, searchFields)
		}
	}

	return qbuilder.GetQuery()
}

func (qb *QueryBuilder) ExecuteAndFilter(
	ctx context.Context,
	query *bun.SelectQuery,
) (*temporaltype.QueryExecutionResult, error) {
	var rows []map[string]any
	err := query.Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(rows) == 0 {
		return &temporaltype.QueryExecutionResult{
			Columns: []string{},
			Rows:    []map[string]any{},
			Total:   0,
		}, nil
	}

	// Collect all columns from first row
	allColumns := make([]string, 0, len(rows[0]))
	for col := range rows[0] {
		allColumns = append(allColumns, col)
	}

	// Filter out excluded columns using reportuils
	columns := reportuils.FilterColumns(allColumns)
	filteredRows := reportuils.FilterRowData(rows, columns)

	return &temporaltype.QueryExecutionResult{
		Columns: columns,
		Rows:    filteredRows,
		Total:   len(rows),
	}, nil
}

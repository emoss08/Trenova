package dbhelper

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

type CursorListParams[T any] struct {
	Query      func(items *[]T) *bun.SelectQuery
	Filter     *pagination.QueryOptions
	Cursor     pagination.CursorInfo
	TotalCount *int
	Apply      func(*bun.SelectQuery) (*bun.SelectQuery, error)
}

func CursorList[T any](
	ctx context.Context,
	params CursorListParams[T],
) (*pagination.CursorListResult[T], error) {
	if params.Query == nil {
		return nil, errors.New("cursor list query builder is required")
	}

	limit := cursorListLimit(params.Filter, params.Cursor)

	items := make([]T, 0, limit+1)
	query := params.Query(&items)
	if query == nil {
		return nil, errors.New("cursor list query builder returned nil")
	}

	if params.Apply != nil {
		var err error
		query, err = params.Apply(query)
		if err != nil {
			return nil, err
		}
		if query == nil {
			return nil, errors.New("cursor list apply returned nil")
		}
	}

	cursorColumns := cursorListColumns(params.Filter)
	for _, column := range cursorColumns {
		query = query.ColumnExpr("? AS ?", bun.Safe(column.SQLExpression), bun.Ident(column.Alias))
	}

	if err := query.Limit(limit + 1).Scan(ctx); err != nil {
		return nil, err
	}

	cursorValues, err := cursorValuesFromItems(items, len(cursorColumns))
	if err != nil {
		return nil, err
	}

	result := pagination.NewCursorListResultWithTotalCount(items, limit, params.TotalCount)
	if params.Filter != nil && len(params.Filter.CursorSort) > 0 {
		result.WithCursorSort(params.Filter.CursorSort)
	}
	if len(cursorValues) > 0 {
		if err := result.WithCursorValues(cursorValues); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func cursorValuesFromItems[T any](items []T, count int) ([][]any, error) {
	if count == 0 {
		return nil, nil
	}
	values := make([][]any, 0, len(items))
	for _, item := range items {
		carrier, ok := any(item).(pagination.CursorValueCarrier)
		if !ok {
			return nil, errors.New("cursor list item does not expose cursor values")
		}
		rowValues := carrier.CursorValues(count)
		if len(rowValues) != count {
			return nil, errors.New("cursor list item cursor value count is invalid")
		}
		values = append(values, rowValues)
	}

	return values, nil
}

func cursorListColumns(filter *pagination.QueryOptions) []pagination.CursorValueColumn {
	if filter == nil || len(filter.CursorColumns) == 0 {
		return nil
	}

	return filter.CursorColumns
}

func cursorListLimit(filter *pagination.QueryOptions, cursor pagination.CursorInfo) int {
	if cursor.Limit > 0 {
		return cursor.Limit
	}
	if filter != nil {
		return filter.Pagination.SafeLimit()
	}

	return pagination.DefaultLimit
}

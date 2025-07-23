package common

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"
)

// ExecuteListQuery performs a standard list query with count
func ExecuteListQuery[T any](
	ctx context.Context,
	query *bun.SelectQuery,
) (*ports.ListResult[T], error) {
	var items []T
	count, err := query.ScanAndCount(ctx, &items)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = make([]T, 0)
	}

	return &ports.ListResult[T]{
		Items: items,
		Total: count,
	}, nil
}

// ListWithFallback executes a list query with empty slice fallback
func ListWithFallback[T any](
	ctx context.Context,
	query *bun.SelectQuery,
) ([]T, error) {
	var items []T
	err := query.Scan(ctx, &items)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = make([]T, 0)
	}

	return items, nil
}

// ApplyDefaultListOrdering adds default ordering to a query
func ApplyDefaultListOrdering(
	query *bun.SelectQuery,
	tableAlias string,
	additionalOrders ...string,
) *bun.SelectQuery {
	orders := append(additionalOrders, tableAlias+".created_at DESC")
	return query.Order(orders...)
}

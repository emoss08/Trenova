package driverportalresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
)

func settlementDisputeConnectionToModel(
	result *pagination.CursorListResult[*driversettlement.Dispute],
) (*gqlmodel.SettlementDisputeConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *driversettlement.Dispute, cursor string) *gqlmodel.SettlementDisputeEdge {
			return &gqlmodel.SettlementDisputeEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.SettlementDisputeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.SettlementDisputeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func driverExpenseConnectionToModel(
	result *pagination.CursorListResult[*driverpay.Expense],
) (*gqlmodel.DriverExpenseConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *driverpay.Expense, cursor string) *gqlmodel.DriverExpenseEdge {
			return &gqlmodel.DriverExpenseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.DriverExpenseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.DriverExpenseConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

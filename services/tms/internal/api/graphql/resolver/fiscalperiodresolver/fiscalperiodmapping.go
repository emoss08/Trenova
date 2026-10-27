package fiscalperiodresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/pkg/pagination"
)

func fiscalPeriodColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FiscalPeriodSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func fiscalPeriodConnectionToModel(
	result *pagination.CursorListResult[*fiscalperiod.FiscalPeriod],
) (*gqlmodel.FiscalPeriodConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *fiscalperiod.FiscalPeriod, cursor string) *gqlmodel.FiscalPeriodEdge {
			return &gqlmodel.FiscalPeriodEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FiscalPeriodEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FiscalPeriodConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

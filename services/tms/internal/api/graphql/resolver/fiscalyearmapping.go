package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/pagination"
)

func fiscalYearColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FiscalYearSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func fiscalYearConnectionToModel(
	result *pagination.CursorListResult[*fiscalyear.FiscalYear],
) (*gqlmodel.FiscalYearConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *fiscalyear.FiscalYear, cursor string) *gqlmodel.FiscalYearEdge {
			return &gqlmodel.FiscalYearEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FiscalYearEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FiscalYearConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

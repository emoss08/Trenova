package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/pkg/pagination"
)

func rateTableColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.RateTableSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func rateTableConnectionToModel(
	result *pagination.CursorListResult[*ratetable.RateTable],
) (*gqlmodel.RateTableConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *ratetable.RateTable, cursor string) *gqlmodel.RateTableEdge {
			return &gqlmodel.RateTableEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.RateTableEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateTableConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

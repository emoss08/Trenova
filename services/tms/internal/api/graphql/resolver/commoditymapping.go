package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/pkg/pagination"
)

func commodityColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.CommoditySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func commodityConnectionToModel(
	result *pagination.CursorListResult[*commodity.Commodity],
) (*gqlmodel.CommodityConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *commodity.Commodity, cursor string) *gqlmodel.CommodityEdge {
			return &gqlmodel.CommodityEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CommodityEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CommodityConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

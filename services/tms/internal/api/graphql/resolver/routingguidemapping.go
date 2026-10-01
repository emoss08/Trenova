package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
)

func routingGuideColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.RoutingGuideSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func routingGuideConnectionToModel(
	result *pagination.CursorListResult[*tender.RoutingGuide],
) (*gqlmodel.RoutingGuideConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *tender.RoutingGuide, cursor string) *gqlmodel.RoutingGuideEdge {
			return &gqlmodel.RoutingGuideEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.RoutingGuideEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RoutingGuideConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

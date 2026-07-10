package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/pagination"
)

func locationColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.LocationSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func locationConnectionToModel(
	result *pagination.CursorListResult[*location.Location],
) (*gqlmodel.LocationConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *location.Location, cursor string) *gqlmodel.LocationEdge {
			return &gqlmodel.LocationEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.LocationEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.LocationConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

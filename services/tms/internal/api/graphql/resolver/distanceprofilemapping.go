package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/pagination"
)

func distanceProfileColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DistanceProfileSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func distanceProfileConnectionToModel(
	result *pagination.CursorListResult[*distanceprofile.DistanceProfile],
) (*gqlmodel.DistanceProfileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *distanceprofile.DistanceProfile, cursor string) *gqlmodel.DistanceProfileEdge {
			return &gqlmodel.DistanceProfileEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DistanceProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DistanceProfileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

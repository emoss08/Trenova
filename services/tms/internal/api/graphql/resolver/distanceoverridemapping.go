package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/pkg/pagination"
)

func distanceOverrideConnectionToModel(
	result *pagination.CursorListResult[*distanceoverride.DistanceOverride],
) (*gqlmodel.DistanceOverrideConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *distanceoverride.DistanceOverride, cursor string) *gqlmodel.DistanceOverrideEdge {
			return &gqlmodel.DistanceOverrideEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DistanceOverrideEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DistanceOverrideConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

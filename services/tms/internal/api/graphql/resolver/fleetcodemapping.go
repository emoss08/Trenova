package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/pagination"
)

func fleetCodeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FleetCodeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func fleetCodeConnectionToModel(
	result *pagination.CursorListResult[*fleetcode.FleetCode],
) (*gqlmodel.FleetCodeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *fleetcode.FleetCode, cursor string) *gqlmodel.FleetCodeEdge {
			return &gqlmodel.FleetCodeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FleetCodeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FleetCodeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

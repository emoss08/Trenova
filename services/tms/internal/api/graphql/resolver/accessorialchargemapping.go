package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
)

func accessorialChargeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AccessorialChargeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func accessorialChargeConnectionToModel(
	result *pagination.CursorListResult[*accessorialcharge.AccessorialCharge],
) (*gqlmodel.AccessorialChargeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *accessorialcharge.AccessorialCharge, cursor string) *gqlmodel.AccessorialChargeEdge {
			return &gqlmodel.AccessorialChargeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AccessorialChargeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccessorialChargeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

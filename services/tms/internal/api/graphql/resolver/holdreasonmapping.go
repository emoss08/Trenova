package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/pkg/pagination"
)

func holdReasonColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HoldReasonSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func holdReasonConnectionToModel(
	result *pagination.CursorListResult[*holdreason.HoldReason],
) (*gqlmodel.HoldReasonConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *holdreason.HoldReason, cursor string) *gqlmodel.HoldReasonEdge {
			return &gqlmodel.HoldReasonEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HoldReasonEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HoldReasonConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

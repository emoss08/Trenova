package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/pkg/pagination"
)

func serviceTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.ServiceTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func serviceTypeConnectionToModel(
	result *pagination.CursorListResult[*servicetype.ServiceType],
) (*gqlmodel.ServiceTypeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *servicetype.ServiceType, cursor string) *gqlmodel.ServiceTypeEdge {
			return &gqlmodel.ServiceTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ServiceTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ServiceTypeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

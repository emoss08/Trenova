package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/pagination"
)

func locationCategoryColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.LocationCategorySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func locationCategoryConnectionToModel(
	result *pagination.CursorListResult[*locationcategory.LocationCategory],
) (*gqlmodel.LocationCategoryConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *locationcategory.LocationCategory, cursor string) *gqlmodel.LocationCategoryEdge {
			return &gqlmodel.LocationCategoryEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.LocationCategoryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.LocationCategoryConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

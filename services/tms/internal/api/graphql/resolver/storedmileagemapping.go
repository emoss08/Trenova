package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/pkg/pagination"
)

func storedMileageColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.StoredMileageSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func storedMileageConnectionToModel(
	result *pagination.CursorListResult[*storedmileage.StoredMileage],
) (*gqlmodel.StoredMileageConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *storedmileage.StoredMileage, cursor string) *gqlmodel.StoredMileageEdge {
			return &gqlmodel.StoredMileageEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.StoredMileageEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.StoredMileageConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

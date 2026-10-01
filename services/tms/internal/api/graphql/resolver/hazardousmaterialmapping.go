package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/pagination"
)

func hazardousMaterialColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HazardousMaterialSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func hazardousMaterialConnectionToModel(
	result *pagination.CursorListResult[*hazardousmaterial.HazardousMaterial],
) (*gqlmodel.HazardousMaterialConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *hazardousmaterial.HazardousMaterial, cursor string) *gqlmodel.HazardousMaterialEdge {
			return &gqlmodel.HazardousMaterialEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HazardousMaterialEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HazardousMaterialConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

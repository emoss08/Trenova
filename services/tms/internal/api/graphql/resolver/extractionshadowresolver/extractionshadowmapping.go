package extractionshadowresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow"
	"github.com/emoss08/trenova/pkg/pagination"
)

func extractionShadowResultColumns(ctx context.Context, nodePathPrefix string) []string {
	return gqlconnection.ProjectedSelection(
		ctx,
		projection.ExtractionShadowResultSpec,
		nodePathPrefix,
	).Columns
}

func extractionShadowResultConnectionToModel(
	result *pagination.CursorListResult[*extractionshadow.ShadowResult],
) (*gqlmodel.ExtractionShadowResultConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *extractionshadow.ShadowResult, cursor string) *gqlmodel.ExtractionShadowResultEdge {
			return &gqlmodel.ExtractionShadowResultEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.ExtractionShadowResultEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ExtractionShadowResultConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

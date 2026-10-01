package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/pkg/pagination"
)

func documentTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.DocumentTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func documentTypeConnectionToModel(
	result *pagination.CursorListResult[*documenttype.DocumentType],
) (*gqlmodel.DocumentTypeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *documenttype.DocumentType, cursor string) *gqlmodel.DocumentTypeEdge {
			return &gqlmodel.DocumentTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.DocumentTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.DocumentTypeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

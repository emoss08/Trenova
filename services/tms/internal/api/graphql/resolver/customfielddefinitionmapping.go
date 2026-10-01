package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/pkg/pagination"
)

func customFieldDefinitionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.CustomFieldDefinitionSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func customFieldDefinitionConnectionToModel(
	result *pagination.CursorListResult[*customfield.CustomFieldDefinition],
) (*gqlmodel.CustomFieldDefinitionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *customfield.CustomFieldDefinition, cursor string) *gqlmodel.CustomFieldDefinitionEdge {
			return &gqlmodel.CustomFieldDefinitionEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.CustomFieldDefinitionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CustomFieldDefinitionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

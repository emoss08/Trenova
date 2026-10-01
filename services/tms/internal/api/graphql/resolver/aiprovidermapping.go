package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
)

func aiProviderColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AIProviderSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	columns := selection.Columns
	if len(columns) > 0 && graphql.FieldRequested(ctx, nodePathPrefix+".hasApiKey") {
		columns = append(columns, buncolgen.ProviderColumns.APIKey.Name)
	}

	return columns
}

func aiProviderConnectionToModel(
	result *pagination.CursorListResult[*aiprovider.Provider],
) (*gqlmodel.AIProviderConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *aiprovider.Provider, cursor string) *gqlmodel.AIProviderEdge {
			return &gqlmodel.AIProviderEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AIProviderEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AIProviderConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

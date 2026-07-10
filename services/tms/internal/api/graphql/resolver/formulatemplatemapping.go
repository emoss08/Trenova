package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
)

func formulaTemplateColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FormulaTemplateSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func formulaTemplateConnectionToModel(
	result *pagination.CursorListResult[*formulatemplate.FormulaTemplate],
) (*gqlmodel.FormulaTemplateConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *formulatemplate.FormulaTemplate, cursor string) *gqlmodel.FormulaTemplateEdge {
			return &gqlmodel.FormulaTemplateEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FormulaTemplateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FormulaTemplateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

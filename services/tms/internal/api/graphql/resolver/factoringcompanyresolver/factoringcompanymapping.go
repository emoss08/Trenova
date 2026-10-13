package factoringcompanyresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/pagination"
)

func factoringCompanyColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FactoringCompanySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func factoringCompanyConnectionToModel(
	result *pagination.CursorListResult[*factoringcompany.FactoringCompany],
) (*gqlmodel.FactoringCompanyConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoringcompany.FactoringCompany, cursor string) *gqlmodel.FactoringCompanyEdge {
			return &gqlmodel.FactoringCompanyEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FactoringCompanyEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FactoringCompanyConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

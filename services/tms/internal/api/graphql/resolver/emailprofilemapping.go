package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/pkg/pagination"
)

func emailProfileColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.EmailProfileSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func emailProfileConnectionToModel(
	result *pagination.CursorListResult[*email.Profile],
) (*gqlmodel.EmailProfileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *email.Profile, cursor string) *gqlmodel.EmailProfileEdge {
			return &gqlmodel.EmailProfileEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.EmailProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EmailProfileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

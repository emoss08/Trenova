package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/pkg/pagination"
)

func accountTypeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AccountTypeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func accountTypeConnectionToModel(
	result *pagination.CursorListResult[*accounttype.AccountType],
) (*gqlmodel.AccountTypeConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *accounttype.AccountType, cursor string) *gqlmodel.AccountTypeEdge {
			return &gqlmodel.AccountTypeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AccountTypeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccountTypeConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

func userConnectionToModel(
	result *pagination.CursorListResult[*tenant.User],
) (*gqlmodel.UserConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *tenant.User, cursor string) *gqlmodel.UserEdge {
			return &gqlmodel.UserEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.UserEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.UserConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

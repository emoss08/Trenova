package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
)

func roleConnectionToModel(
	result *pagination.CursorListResult[*permission.Role],
) (*gqlmodel.RoleConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *permission.Role, cursor string) *gqlmodel.RoleEdge {
			return &gqlmodel.RoleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.RoleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RoleConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

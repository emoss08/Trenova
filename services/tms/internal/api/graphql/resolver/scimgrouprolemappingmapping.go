package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/pkg/pagination"
)

func scimGroupRoleMappingConnectionToModel(
	result *pagination.CursorListResult[*iam.SCIMGroupRoleMapping],
) (*gqlmodel.SCIMGroupRoleMappingConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *iam.SCIMGroupRoleMapping, cursor string) *gqlmodel.SCIMGroupRoleMappingEdge {
			return &gqlmodel.SCIMGroupRoleMappingEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.SCIMGroupRoleMappingEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.SCIMGroupRoleMappingConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

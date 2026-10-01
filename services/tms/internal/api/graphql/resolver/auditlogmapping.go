package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
)

func auditEntryConnectionToModel(
	result *pagination.CursorListResult[*audit.Entry],
) (*gqlmodel.AuditEntryConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *audit.Entry, cursor string) *gqlmodel.AuditEntryEdge {
			return &gqlmodel.AuditEntryEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AuditEntryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AuditEntryConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

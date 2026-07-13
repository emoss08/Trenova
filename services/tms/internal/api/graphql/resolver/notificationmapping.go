package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
)

func notificationConnectionToModel(
	result *pagination.CursorListResult[*notification.Notification],
) (*gqlmodel.NotificationConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *notification.Notification, cursor string) *gqlmodel.NotificationEdge {
			return &gqlmodel.NotificationEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.NotificationEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.NotificationConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

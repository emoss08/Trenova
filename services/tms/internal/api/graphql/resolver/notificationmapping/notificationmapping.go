package notificationmapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func ApplyNotificationFilter(
	req *notificationports.ListNotificationConnectionRequest,
	filter *gqlmodel.NotificationFilterInput,
) {
	if filter == nil {
		return
	}

	if filter.State != nil {
		if state, err := notification.StateFromString(string(*filter.State)); err == nil {
			req.State = state
		}
	}

	if filter.UnreadOnly != nil {
		req.UnreadOnly = *filter.UnreadOnly
	}
}

func NotificationConnectionToModel(
	result *pagination.CursorListResult[*notification.Notification],
) (*gqlmodel.NotificationConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

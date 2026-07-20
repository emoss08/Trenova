package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *mutationResolver) notificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, repositories.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return false, err
	}

	notificationIDs, err := parseIDs(ids)
	if err != nil {
		return false, err
	}

	if err = action(ctx, repositories.NotificationActionRequest{
		IDs:        notificationIDs,
		TenantInfo: tenantInfo(authCtx),
	}); err != nil {
		return false, err
	}

	return true, nil
}

func applyNotificationFilter(
	req *repositories.ListNotificationConnectionRequest,
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

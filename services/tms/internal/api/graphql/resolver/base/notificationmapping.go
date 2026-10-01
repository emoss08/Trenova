package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type NotificationActionParams struct {
	IDs          []string
	TenantInfo   pagination.TenantInfo
	PersonalOnly bool
	Action       func(context.Context, repositories.NotificationActionRequest) error
}

func (r *Resolver) ApplyNotificationAction(
	ctx context.Context,
	params *NotificationActionParams,
) (bool, error) {
	notificationIDs, err := ParseIDs(params.IDs)
	if err != nil {
		return false, err
	}

	if err = params.Action(ctx, repositories.NotificationActionRequest{
		IDs:          notificationIDs,
		TenantInfo:   params.TenantInfo,
		PersonalOnly: params.PersonalOnly,
	}); err != nil {
		return false, err
	}

	return true, nil
}

func ApplyNotificationFilter(
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

func NotificationConnectionToModel(
	result *pagination.CursorListResult[*notification.Notification],
) (*gqlmodel.NotificationConnection, error) {
	page, err := EntityCursorConnection(
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

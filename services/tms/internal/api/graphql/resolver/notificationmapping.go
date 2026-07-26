package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type notificationActionParams struct {
	IDs          []string
	TenantInfo   pagination.TenantInfo
	PersonalOnly bool
	Action       func(context.Context, repositories.NotificationActionRequest) error
}

func (r *mutationResolver) notificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, repositories.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return false, err
	}

	return r.applyNotificationAction(ctx, notificationActionParams{
		IDs:          ids,
		TenantInfo:   tenantInfo(authCtx),
		PersonalOnly: authCtx.IsPortalUser,
		Action:       action,
	})
}

// myNotificationAction resolves the signed-in driver before acting and always
// forces personal scope, so a portal caller can never mutate an
// organization-wide notification even if the session predates portal tagging.
func (r *mutationResolver) myNotificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, repositories.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.requireAuth(ctx)
	if err != nil {
		return false, err
	}

	ti := tenantInfo(authCtx)
	if _, err = r.driverPortalService.ResolveWorker(ctx, ti); err != nil {
		return false, err
	}

	return r.applyNotificationAction(ctx, notificationActionParams{
		IDs:          ids,
		TenantInfo:   ti,
		PersonalOnly: true,
		Action:       action,
	})
}

func (r *mutationResolver) applyNotificationAction(
	ctx context.Context,
	params notificationActionParams,
) (bool, error) {
	notificationIDs, err := parseIDs(params.IDs)
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

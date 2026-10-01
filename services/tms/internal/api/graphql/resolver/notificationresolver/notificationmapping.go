package notificationresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

func (r *MutationResolver) notificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, repositories.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return false, err
	}

	return r.ApplyNotificationAction(ctx, &base.NotificationActionParams{
		IDs:          ids,
		TenantInfo:   base.TenantInfo(authCtx),
		PersonalOnly: authCtx.IsPortalUser,
		Action:       action,
	})
}

package notificationresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
)

func (r *MutationResolver) notificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, notificationports.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return false, err
	}

	return r.ApplyNotificationAction(ctx, &resolvercore.NotificationActionParams{
		IDs:          ids,
		TenantInfo:   resolvercore.TenantInfo(authCtx),
		PersonalOnly: authCtx.IsPortalUser,
		Action:       action,
	})
}

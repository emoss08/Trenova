package driverportalresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
)

// myNotificationAction resolves the signed-in driver before acting and always
// forces personal scope, so a portal caller can never mutate an
// organization-wide notification even if the session predates portal tagging.
func (r *MutationResolver) myNotificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, notificationports.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return false, err
	}

	ti := resolvercore.TenantInfo(authCtx)
	if _, err = r.DriverPortalService.ResolveWorker(ctx, ti); err != nil {
		return false, err
	}

	return r.ApplyNotificationAction(ctx, &resolvercore.NotificationActionParams{
		IDs:          ids,
		TenantInfo:   ti,
		PersonalOnly: true,
		Action:       action,
	})
}

package driverportalresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

// myNotificationAction resolves the signed-in driver before acting and always
// forces personal scope, so a portal caller can never mutate an
// organization-wide notification even if the session predates portal tagging.
func (r *MutationResolver) myNotificationAction(
	ctx context.Context,
	ids []string,
	action func(context.Context, repositories.NotificationActionRequest) error,
) (bool, error) {
	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return false, err
	}

	ti := base.TenantInfo(authCtx)
	if _, err = r.DriverPortalService.ResolveWorker(ctx, ti); err != nil {
		return false, err
	}

	return r.ApplyNotificationAction(ctx, &base.NotificationActionParams{
		IDs:          ids,
		TenantInfo:   ti,
		PersonalOnly: true,
		Action:       action,
	})
}

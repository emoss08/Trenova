package resolvercore

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type NotificationActionParams struct {
	IDs          []string
	TenantInfo   pagination.TenantInfo
	PersonalOnly bool
	Action       func(context.Context, notificationports.NotificationActionRequest) error
}

func (r *Core) ApplyNotificationAction(
	ctx context.Context,
	params *NotificationActionParams,
) (bool, error) {
	notificationIDs, err := gqlid.ParseIDs(params.IDs)
	if err != nil {
		return false, err
	}

	if err = params.Action(ctx, notificationports.NotificationActionRequest{
		IDs:          notificationIDs,
		TenantInfo:   params.TenantInfo,
		PersonalOnly: params.PersonalOnly,
	}); err != nil {
		return false, err
	}

	return true, nil
}

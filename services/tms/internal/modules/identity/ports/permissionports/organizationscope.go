package permissionports

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func RequireOrganizationScope(
	ctx context.Context,
	engine PermissionEngine,
	check *PermissionCheckRequest,
) error {
	result, err := engine.Check(ctx, check)
	if err != nil {
		return err
	}
	if result == nil || !result.Allowed {
		return errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}",
			check.Resource,
			check.Operation,
		)
	}
	if result.DataScope.IsNarrow() {
		return errortypes.NewAuthorizationError(
			"This covers the whole organization, and your access is limited to part of it",
		)
	}

	return nil
}

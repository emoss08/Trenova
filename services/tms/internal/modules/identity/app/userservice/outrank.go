package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) requireActorCoversTarget(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	target *tenant.User,
) error {
	if target == nil || target.ID == tenantInfo.UserID {
		return nil
	}

	for _, membership := range target.Memberships {
		if membership == nil || membership.OrganizationID == tenantInfo.OrgID {
			continue
		}
		allowed, err := s.mayManageUsersIn(
			ctx,
			tenantInfo.UserID,
			membership.OrganizationID,
			membership.BusinessUnitID,
		)
		if err != nil {
			return err
		}
		if !allowed {
			return errTargetOutranksActor()
		}
	}

	held, err := s.permissions.GetEffectivePermissions(ctx, tenantInfo.UserID, tenantInfo.OrgID)
	if err != nil {
		return err
	}
	theirs, err := s.permissions.GetEffectivePermissions(ctx, target.ID, tenantInfo.OrgID)
	if err != nil {
		return err
	}
	if !held.MaxSensitivity.CanAccess(theirs.MaxSensitivity) {
		return errTargetOutranksActor()
	}
	for resource, grant := range theirs.Resources {
		if _, violated := held.DelegationViolation(
			resource,
			grant.Operations,
			grant.DataScope,
		); violated {
			return errTargetOutranksActor()
		}
	}

	return nil
}

func errTargetOutranksActor() error {
	return errortypes.NewAuthorizationError(
		"You can change another person's sign-in only when your own access covers everything theirs does",
	)
}

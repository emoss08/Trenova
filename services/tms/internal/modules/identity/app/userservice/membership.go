package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type membershipAuthorization struct {
	actorID        pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	previous       []*tenant.OrganizationMembership
	desired        []pulid.ID
}

func (s *Service) authorizeMembershipChange(
	ctx context.Context,
	req *membershipAuthorization,
) error {
	for _, orgID := range changedOrganizations(req.previous, req.desired) {
		if orgID == req.organizationID {
			continue
		}

		allowed, err := s.mayManageUsersIn(ctx, req.actorID, orgID, req.businessUnitID)
		if err != nil {
			return err
		}
		if !allowed {
			return errortypes.NewAuthorizationError(
				"You need permission to manage users in every organization you add or remove",
			)
		}
	}

	return nil
}

func (s *Service) mayManageUsersIn(
	ctx context.Context,
	actorID, orgID, buID pulid.ID,
) (bool, error) {
	checkCtx := authctx.WithoutSessionRoleActivation(dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		UserID:         actorID,
	}))

	result, err := s.permissions.Check(checkCtx, &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    actorID,
		UserID:         actorID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Resource:       permission.ResourceUser.String(),
		Operation:      permission.OpUpdate,
	})
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

func changedOrganizations(
	previous []*tenant.OrganizationMembership,
	desired []pulid.ID,
) []pulid.ID {
	before := make(map[pulid.ID]struct{}, len(previous))
	for _, membership := range previous {
		before[membership.OrganizationID] = struct{}{}
	}
	after := make(map[pulid.ID]struct{}, len(desired))
	for _, orgID := range desired {
		after[orgID] = struct{}{}
	}

	changed := make([]pulid.ID, 0, len(before)+len(after))
	for _, membership := range previous {
		if _, kept := after[membership.OrganizationID]; !kept {
			changed = append(changed, membership.OrganizationID)
		}
	}
	for _, orgID := range desired {
		if _, existed := before[orgID]; existed {
			continue
		}
		before[orgID] = struct{}{}
		changed = append(changed, orgID)
	}

	return changed
}

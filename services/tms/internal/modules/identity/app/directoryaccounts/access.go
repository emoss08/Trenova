package directoryaccounts

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccessRequest struct {
	Actor  Actor
	UserID pulid.ID
}

func (s *Service) Revoke(ctx context.Context, req *AccessRequest) error {
	user, err := s.member(ctx, &req.Actor, req.UserID)
	if err != nil {
		if isMissing(err) {
			return nil
		}
		return err
	}

	memberships, err := s.users.ListOrganizationMemberships(
		ctx,
		req.UserID,
		req.Actor.BusinessUnitID,
	)
	if err != nil {
		return err
	}

	if memberOf(memberships, req.Actor.OrganizationID) &&
		!coveredBy(memberships, req.Actor.OrganizationID) {
		return s.lifecycle.ExitOrganization(ctx, &userservice.OrganizationExit{
			Actor:          permissionports.SystemAuditActor(),
			Metadata:       req.Actor.Metadata(),
			UserID:         req.UserID,
			OrganizationID: req.Actor.OrganizationID,
			BusinessUnitID: req.Actor.BusinessUnitID,
		})
	}

	if !solelyIn(user, memberships, req.Actor.OrganizationID) || !user.IsActive() {
		return nil
	}

	updated, err := s.setStatus(ctx, &req.Actor, user, domaintypes.StatusInactive)
	if err != nil {
		return err
	}

	return s.lifecycle.EndUserAccess(ctx, req.Actor.TenantInfo(), updated.ID)
}

func (s *Service) Restore(ctx context.Context, req *AccessRequest) error {
	memberships, err := s.users.ListOrganizationMemberships(
		ctx,
		req.UserID,
		req.Actor.BusinessUnitID,
	)
	if err != nil {
		return err
	}

	if !memberOf(memberships, req.Actor.OrganizationID) {
		previous := memberships
		organizationIDs := make([]pulid.ID, 0, len(memberships)+1)
		for _, membership := range memberships {
			organizationIDs = append(organizationIDs, membership.OrganizationID)
		}
		organizationIDs = append(organizationIDs, req.Actor.OrganizationID)

		memberships, err = s.users.ReplaceOrganizationMemberships(
			ctx,
			tenantports.ReplaceOrganizationMembershipsRequest{
				UserID:          req.UserID,
				BusinessUnitID:  req.Actor.BusinessUnitID,
				OrganizationIDs: organizationIDs,
			},
		)
		if err != nil {
			return err
		}
		s.recordMembershipChange(ctx, &req.Actor, req.UserID, previous, memberships)
	}

	user, err := s.member(ctx, &req.Actor, req.UserID)
	if err != nil {
		return err
	}
	if !solelyIn(user, memberships, req.Actor.OrganizationID) || user.IsActive() {
		return nil
	}

	_, err = s.setStatus(ctx, &req.Actor, user, domaintypes.StatusActive)
	return err
}

func memberOf(memberships []*tenant.OrganizationMembership, organizationID pulid.ID) bool {
	for _, membership := range memberships {
		if membership.OrganizationID == organizationID {
			return true
		}
	}

	return false
}

func (s *Service) recordMembershipChange(
	ctx context.Context,
	actor *Actor,
	userID pulid.ID,
	before, after []*tenant.OrganizationMembership,
) {
	s.recordChange(ctx, actor, &permissionports.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: userID.String(),
		Operation:  permission.OpUpdate,
		Before:     map[string]any{"organizationIds": organizationIDs(before)},
		After:      map[string]any{"organizationIds": organizationIDs(after)},
		Comment:    "Organization memberships changed",
	})
}

func organizationIDs(memberships []*tenant.OrganizationMembership) []string {
	ids := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		ids = append(ids, membership.OrganizationID.String())
	}

	return ids
}

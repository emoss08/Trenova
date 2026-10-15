package userservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	revokedAllSessions          = "all"
	revokedOrganizationSessions = "organization"
)

func lostSignInAccess(before, after *tenant.User) bool {
	if after.ValidateStatus() == nil {
		return false
	}
	return before.Status != after.Status || before.IsLocked != after.IsLocked
}

func (s *Service) endAllSessions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	if err := s.sr.DeleteAllForUser(ctx, userID); err != nil {
		s.l.Error(
			"failed to end the sessions of a user who lost access",
			zap.String("userID", userID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("end sessions for user %s: %w", userID, err)
	}

	s.recordSessionsEnded(ctx, tenantInfo, userID, revokedAllSessions)
	return nil
}

func (s *Service) endOrganizationSessions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	removed map[pulid.ID]struct{},
) error {
	sessions, err := s.sr.ListForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list sessions for user %s: %w", userID, err)
	}

	var errs []error
	ended := 0
	for _, sess := range sessions {
		if _, ok := removed[sess.OrganizationID]; !ok {
			continue
		}
		if err = s.sr.Delete(ctx, sess.ID); err != nil {
			errs = append(errs, fmt.Errorf("end session %s: %w", sess.ID, err))
			continue
		}
		ended++
	}

	if ended > 0 {
		s.recordSessionsEnded(ctx, tenantInfo, userID, revokedOrganizationSessions)
	}

	return errors.Join(errs...)
}

func (s *Service) revokeRemovedOrganizations(
	ctx context.Context,
	change *membershipChange,
) error {
	removed := removedOrganizations(change.before, change.after)
	if len(removed) == 0 {
		return nil
	}

	errs := make([]error, 0, len(removed)+1)
	for orgID := range removed {
		if err := s.roles.RevokeOrganizationRoles(ctx, roleservice.RevokeOrganizationRolesRequest{
			ActorID:        change.actorID,
			UserID:         change.userID,
			OrganizationID: orgID,
			BusinessUnitID: change.businessUnitID,
		}); err != nil {
			errs = append(errs, fmt.Errorf("revoke roles in organization %s: %w", orgID, err))
		}
	}

	if err := s.endOrganizationSessions(ctx, pagination.TenantInfo{
		OrgID:  change.organizationID,
		BuID:   change.businessUnitID,
		UserID: change.actorID,
	}, change.userID, removed); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func removedOrganizations(
	before, after []*tenant.OrganizationMembership,
) map[pulid.ID]struct{} {
	kept := make(map[pulid.ID]struct{}, len(after))
	for _, membership := range after {
		kept[membership.OrganizationID] = struct{}{}
	}

	removed := make(map[pulid.ID]struct{})
	for _, membership := range before {
		if _, ok := kept[membership.OrganizationID]; !ok {
			removed[membership.OrganizationID] = struct{}{}
		}
	}
	return removed
}

func (s *Service) recordSessionsEnded(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	scope string,
) {
	if s.authEvents == nil {
		return
	}
	s.authEvents.Record(ctx, &iamports.AuthEventRecord{
		Provider:       iamports.AuthEventProviderSessionRevoked,
		Outcome:        iam.AuthEventOutcomeSuccess,
		UserID:         userID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		MFAState:       scope,
	})
}

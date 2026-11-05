package userservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
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
	return errors.Join(
		s.deleteAllSessions(ctx, tenantInfo, userID),
		s.revokeLongLivedAccess(ctx, tenantInfo, userID),
	)
}

func (s *Service) revokeLongLivedAccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	return s.revokeAccessIn(ctx, tenantInfo, userID, pulid.Nil)
}

func (s *Service) revokeAccessIn(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID, organizationID pulid.ID,
) error {
	errs := make([]error, 0, len(s.accessRevokers))
	for _, revoker := range s.accessRevokers {
		if err := revoker.RevokeUserAccess(ctx, tenantports.RevokeUserAccessRequest{
			TenantInfo:     tenantInfo,
			UserID:         userID,
			OrganizationID: organizationID,
		}); err != nil {
			s.l.Error(
				"failed to revoke the access of a user who lost sign-in",
				zap.String("userID", userID.String()),
				zap.Error(err),
			)
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *Service) deleteAllSessions(
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

	s.recordSessionsEnded(ctx, tenantInfo, userID, iamports.SessionRevocationAll)
	return nil
}

func (s *Service) endOtherSessions(ctx context.Context, scope *SessionScope) (int, error) {
	sessions, err := s.sr.ListForUser(ctx, scope.TenantInfo.UserID)
	if err != nil {
		return 0, fmt.Errorf("list sessions for user %s: %w", scope.TenantInfo.UserID, err)
	}

	var errs []error
	ended := 0
	for _, sess := range sessions {
		if sess.ID == scope.CurrentSessionID {
			continue
		}
		if err = s.sr.Delete(ctx, sess.ID); err != nil {
			errs = append(errs, fmt.Errorf("end session %s: %w", sess.ID, err))
			continue
		}
		ended++
	}

	return ended, errors.Join(errs...)
}

func (s *Service) endLingeringSessions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	user *tenant.User,
) error {
	if user.ValidateStatus() == nil {
		return nil
	}

	sessions, err := s.sr.ListForUser(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("list sessions for user %s: %w", user.ID, err)
	}
	if len(sessions) == 0 {
		return s.revokeLongLivedAccess(ctx, tenantInfo, user.ID)
	}

	return s.endAllSessions(ctx, tenantInfo, user.ID)
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
		s.recordSessionsEnded(ctx, tenantInfo, userID, iamports.SessionRevocationOrganization)
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

	tenantInfo := pagination.TenantInfo{
		OrgID:  change.organizationID,
		BuID:   change.businessUnitID,
		UserID: change.actorID,
	}
	if err := s.endOrganizationSessions(ctx, tenantInfo, change.userID, removed); err != nil {
		errs = append(errs, err)
	}
	for orgID := range removed {
		if err := s.revokeAccessIn(ctx, tenantInfo, change.userID, orgID); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

const (
	orphanSweepBatchSize  = 500
	orphanSweepMaxBatches = 20
)

type OrganizationAccessSweepResult struct {
	Revoked int `json:"revoked"`
	Failed  int `json:"failed"`
}

func (s *Service) SweepOrphanedOrganizationAccess(
	ctx context.Context,
	now int64,
) (*OrganizationAccessSweepResult, error) {
	result := new(OrganizationAccessSweepResult)
	var errs []error

	for range orphanSweepMaxBatches {
		orphans, err := s.roleRepo.ListOrphanedAssignments(ctx, now, orphanSweepBatchSize)
		if err != nil {
			return result, fmt.Errorf("list orphaned role assignments: %w", err)
		}

		failedBefore := result.Failed
		for _, orphan := range orphans {
			if err = s.revokeOrphanedAccess(ctx, orphan); err != nil {
				result.Failed++
				errs = append(errs, err)
				continue
			}
			result.Revoked++
		}

		if len(orphans) < orphanSweepBatchSize || result.Failed > failedBefore {
			break
		}
	}

	if result.Revoked > 0 || result.Failed > 0 {
		s.l.Info(
			"swept role assignments left in organizations their holders no longer belong to",
			zap.Int("revoked", result.Revoked),
			zap.Int("failed", result.Failed),
		)
	}

	return result, errors.Join(errs...)
}

func (s *Service) revokeOrphanedAccess(
	ctx context.Context,
	orphan permissionports.OrphanedAssignment,
) error {
	if err := s.roles.RevokeOrganizationRoles(ctx, roleservice.RevokeOrganizationRolesRequest{
		UserID:         orphan.UserID,
		OrganizationID: orphan.OrganizationID,
		BusinessUnitID: orphan.BusinessUnitID,
	}); err != nil {
		return fmt.Errorf(
			"revoke roles of user %s in organization %s: %w",
			orphan.UserID,
			orphan.OrganizationID,
			err,
		)
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: orphan.OrganizationID,
		BuID:  orphan.BusinessUnitID,
	}

	return errors.Join(
		s.endOrganizationSessions(
			ctx,
			tenantInfo,
			orphan.UserID,
			map[pulid.ID]struct{}{orphan.OrganizationID: {}},
		),
		s.revokeAccessIn(ctx, tenantInfo, orphan.UserID, orphan.OrganizationID),
	)
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
	s.authEvents.Record(ctx, iamports.SessionRevokedEvent(tenantInfo, userID, scope))
}

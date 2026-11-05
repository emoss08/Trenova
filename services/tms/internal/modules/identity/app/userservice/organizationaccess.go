package userservice

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

type OrganizationExit struct {
	Actor          kernelports.AuditActor
	Metadata       map[string]any
	UserID         pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (s *Service) ExitOrganization(ctx context.Context, req *OrganizationExit) error {
	previous, err := s.repo.ListOrganizationMemberships(ctx, req.UserID, req.BusinessUnitID)
	if err != nil {
		return err
	}

	kept := make([]pulid.ID, 0, len(previous))
	member := false
	for _, membership := range previous {
		if membership.OrganizationID == req.OrganizationID {
			member = true
			continue
		}
		kept = append(kept, membership.OrganizationID)
	}
	if !member {
		return nil
	}
	if len(kept) == 0 {
		return tenant.NewNoOrganizationMembershipsError()
	}

	memberships, err := s.repo.ReplaceOrganizationMemberships(
		ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			UserID:          req.UserID,
			BusinessUnitID:  req.BusinessUnitID,
			OrganizationIDs: kept,
		},
	)
	if err != nil {
		return err
	}

	change := &membershipChange{
		userID:         req.UserID,
		organizationID: req.OrganizationID,
		businessUnitID: req.BusinessUnitID,
		before:         previous,
		after:          memberships,
	}
	s.recordExit(ctx, req, change)

	return s.revokeRemovedOrganizations(ctx, change)
}

func (s *Service) EndUserAccess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	return s.endAllSessions(ctx, tenantInfo, userID)
}

func (s *Service) PublishUserChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	user *tenant.User,
	action string,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		Resource:       permission.ResourceUser.String(),
		Action:         action,
		RecordID:       user.ID,
		Entity:         user,
	}); err != nil {
		s.l.Warn("failed to publish user invalidation", zap.Error(err))
	}
}

func (s *Service) recordExit(ctx context.Context, req *OrganizationExit, change *membershipChange) {
	before := membershipOrganizationIDs(change.before)
	after := membershipOrganizationIDs(change.after)

	metadata := map[string]any{
		"addedOrganizationIds":   sliceutils.Difference(after, before),
		"removedOrganizationIds": sliceutils.Difference(before, after),
	}
	maps.Copy(metadata, req.Metadata)

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     req.UserID.String(),
		Operation:      permission.OpUpdate,
		Actor:          req.Actor,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		Before:         map[string]any{organizationIDsKey: before},
		After:          map[string]any{organizationIDsKey: after},
		Comment:        "Organization memberships changed",
		Metadata:       metadata,
	})
}

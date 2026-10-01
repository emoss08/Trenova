package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

type membershipChange struct {
	actorID        pulid.ID
	userID         pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	before         []*tenant.OrganizationMembership
	after          []*tenant.OrganizationMembership
}

func (s *Service) recordMembershipChange(ctx context.Context, change *membershipChange) {
	if s.auditor == nil {
		return
	}

	before := membershipOrganizationIDs(change.before)
	after := membershipOrganizationIDs(change.after)

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: change.userID.String(),
		Operation:  permission.OpUpdate,
		Actor: services.UserActor(pagination.TenantInfo{
			OrgID:  change.organizationID,
			BuID:   change.businessUnitID,
			UserID: change.actorID,
		}).AuditActor(),
		OrganizationID: change.organizationID,
		BusinessUnitID: change.businessUnitID,
		Before:         map[string]any{"organizationIds": before},
		After:          map[string]any{"organizationIds": after},
		Comment:        "Organization memberships changed",
		Metadata: map[string]any{
			"addedOrganizationIds":   sliceutils.Difference(after, before),
			"removedOrganizationIds": sliceutils.Difference(before, after),
		},
	})
}

func membershipOrganizationIDs(memberships []*tenant.OrganizationMembership) []string {
	ids := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		ids = append(ids, membership.OrganizationID.String())
	}
	return ids
}

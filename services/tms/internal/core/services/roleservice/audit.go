package roleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	metadataUserID       = "userId"
	metadataAssignmentID = "assignmentId"
)

type roleChange struct {
	actorID        pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
	resourceID     string
	operation      permission.Operation
	before         any
	after          any
	comment        string
	metadata       map[string]any
}

func (s *Service) recordChange(ctx context.Context, change *roleChange) {
	if s.auditor == nil {
		return
	}

	var actor services.AuditActor
	if change.actorID.IsNotNil() {
		actor = services.UserActor(pagination.TenantInfo{
			OrgID:  change.organizationID,
			BuID:   change.businessUnitID,
			UserID: change.actorID,
		}).AuditActor()
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
		Resource:       permission.ResourceRole,
		ResourceID:     change.resourceID,
		Operation:      change.operation,
		Actor:          actor,
		OrganizationID: change.organizationID,
		BusinessUnitID: change.businessUnitID,
		Before:         change.before,
		After:          change.after,
		Comment:        change.comment,
		Metadata:       change.metadata,
	})
}

func findResourcePermission(
	perms []*permission.ResourcePermission,
	id pulid.ID,
) *permission.ResourcePermission {
	for _, perm := range perms {
		if perm.ID == id {
			return perm
		}
	}
	return nil
}

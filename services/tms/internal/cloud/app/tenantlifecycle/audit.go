package tenantlifecycle

import (
	"context"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/zap"
)

const (
	keyUserID         = "userId"
	keyRoleID         = "roleId"
	keyOrganizationID = "organizationId"
)

func (s *Service) metadata(t *cloudtenant.Tenant) map[string]any {
	return map[string]any{
		"source":                 auditSource,
		"actor":                  controlPlane,
		"controlPlaneInstanceId": t.InstanceID,
		"revision":               t.Desired.Revision,
	}
}

func (s *Service) auditCreated(
	ctx context.Context,
	t *cloudtenant.Tenant,
	created *tenantports.BootstrapTenantResult,
) {
	org := created.Organization
	owner := created.Owner
	actor := permissionports.SystemAuditActor()
	metadata := s.metadata(t)

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     owner.ID.String(),
		Operation:      permission.OpCreate,
		Action:         "user.created",
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         owner.ID.String(),
			"username":        owner.Username,
			"emailAddress":    owner.EmailAddress,
			"name":            owner.Name,
			"status":          string(owner.Status),
			keyOrganizationID: org.ID.String(),
			"membership":      "default",
		},
		Comment:  "The Trenova control plane created the organization's first administrator",
		Metadata: metadata,
	})
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceRole,
		ResourceID:     created.AdminRoleID.String(),
		Operation:      permission.OpCreate,
		Action:         "role.created",
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyRoleID:         created.AdminRoleID.String(),
			"name":            "Organization Administrator",
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "The Trenova control plane created the Organization Administrator role",
		Metadata: metadata,
	})
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceRole,
		ResourceID:     created.AdminRoleID.String(),
		Operation:      permission.OpAssign,
		Action:         "role.assigned",
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         owner.ID.String(),
			keyRoleID:         created.AdminRoleID.String(),
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "The Trenova control plane granted the first administrator the Organization Administrator role",
		Metadata: metadata,
	})
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     created.SystemActor.String(),
		Operation:      permission.OpCreate,
		Action:         "user.created",
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         created.SystemActor.String(),
			"username":        tenant.SystemActorUsername(org.BusinessUnitID),
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "The Trenova control plane created the system actor background work is written under",
		Metadata: metadata,
	})
	s.l.Info("made a shared-server tenant",
		zap.String("instanceId", t.InstanceID),
		zap.String("organizationId", org.ID.String()),
	)
}

func (s *Service) auditLifecycle(
	ctx context.Context,
	t *cloudtenant.Tenant,
	action string,
	comment string,
) {
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceOrganization,
		ResourceID:     t.OrganizationID.String(),
		Operation:      permission.OpUpdate,
		Action:         action,
		Actor:          permissionports.SystemAuditActor(),
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
		After: map[string]any{
			keyOrganizationID: t.OrganizationID.String(),
			"state":           string(t.State),
		},
		Comment:  comment,
		Metadata: s.metadata(t),
	})
	s.l.Info("changed a shared-server tenant's lifecycle",
		zap.String("instanceId", t.InstanceID),
		zap.String("action", action),
	)
}

package accountguard

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Users       tenantports.UserRepository
	Permissions permissionports.PermissionEngine
	Auditor     permissionports.SecurityAuditor
}

type Guard struct {
	users       tenantports.UserRepository
	permissions permissionports.PermissionEngine
	auditor     permissionports.SecurityAuditor
}

func New(p Params) *Guard {
	return &Guard{users: p.Users, permissions: p.Permissions, auditor: p.Auditor}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	TargetID   pulid.ID
	Action     string
}

func (g *Guard) RequireCovers(ctx context.Context, req *Request) error {
	actorID := req.TenantInfo.UserID
	if actorID.IsNotNil() && actorID == req.TargetID {
		return nil
	}
	if actorID.IsNil() {
		return errNotCovered()
	}

	memberships, err := g.users.ListOrganizationMemberships(
		ctx,
		req.TargetID,
		req.TenantInfo.BuID,
	)
	if err != nil {
		return err
	}

	for _, orgID := range organizationsOf(memberships, req.TenantInfo.OrgID) {
		covered, coverErr := g.covers(ctx, req, orgID)
		if coverErr != nil {
			return coverErr
		}
		if !covered {
			g.recordRefusal(ctx, req, orgID)

			return errNotCovered()
		}
	}

	return nil
}

func errNotCovered() error {
	return errortypes.NewAuthorizationError(
		"You can only do this for someone whose access you also hold",
	)
}

func organizationsOf(
	memberships []*tenant.OrganizationMembership,
	current pulid.ID,
) []pulid.ID {
	orgIDs := make([]pulid.ID, 0, len(memberships)+1)
	orgIDs = append(orgIDs, current)
	for _, membership := range memberships {
		if membership != nil && membership.OrganizationID != current {
			orgIDs = append(orgIDs, membership.OrganizationID)
		}
	}

	return orgIDs
}

func (g *Guard) covers(ctx context.Context, req *Request, orgID pulid.ID) (bool, error) {
	scoped := authctx.WithoutSessionRoleActivation(dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: orgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
	}))

	target, err := g.permissions.GetEffectivePermissions(scoped, req.TargetID, orgID)
	if err != nil {
		return false, err
	}
	actor, err := g.permissions.GetEffectivePermissions(scoped, req.TenantInfo.UserID, orgID)
	if err != nil {
		return false, err
	}

	_, uncovered := actor.Uncovered(target)

	return !uncovered, nil
}

func (g *Guard) recordRefusal(ctx context.Context, req *Request, orgID pulid.ID) {
	g.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:   permission.ResourceUser,
		ResourceID: req.TargetID.String(),
		Operation:  permission.OpUpdate,
		Actor: kernelports.AuditActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   req.TenantInfo.UserID,
			UserID:        req.TenantInfo.UserID,
		},
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Comment:        "Refused a sign-in change on an account beyond the actor's own access",
		Metadata: map[string]any{
			"refused":      req.Action,
			"organization": orgID.String(),
		},
	})
}

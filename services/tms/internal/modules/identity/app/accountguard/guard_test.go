package accountguard

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memberships struct {
	tenantports.UserRepository

	orgIDs []pulid.ID
}

func (m *memberships) ListOrganizationMemberships(
	context.Context,
	pulid.ID,
	pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	listed := make([]*tenant.OrganizationMembership, 0, len(m.orgIDs))
	for _, orgID := range m.orgIDs {
		listed = append(listed, &tenant.OrganizationMembership{OrganizationID: orgID})
	}

	return listed, nil
}

type grants map[pulid.ID]map[pulid.ID]map[string]permissionports.EffectiveResourcePermission

type effective struct {
	permissionports.PermissionEngine

	held grants
}

func (e *effective) GetEffectivePermissions(
	_ context.Context,
	userID, orgID pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	resources := e.held[userID][orgID]
	if resources == nil {
		resources = map[string]permissionports.EffectiveResourcePermission{}
	}

	return &permissionports.EffectivePermissions{Resources: resources}, nil
}

func userAdmin() map[string]permissionports.EffectiveResourcePermission {
	return map[string]permissionports.EffectiveResourcePermission{
		permission.ResourceUser.String(): {
			Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			DataScope:  permission.DataScopeOrganization,
		},
	}
}

func everything() map[string]permissionports.EffectiveResourcePermission {
	held := userAdmin()
	held[permission.ResourceRole.String()] = permissionports.EffectiveResourcePermission{
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeAll,
	}

	return held
}

type fixture struct {
	guard    *Guard
	security *securityaudittest.Recorder
	req      *Request
	org      pulid.ID
	other    pulid.ID
	actor    pulid.ID
	target   pulid.ID
	held     grants
}

func newFixture() *fixture {
	f := &fixture{
		security: &securityaudittest.Recorder{},
		org:      pulid.MustNew("org_"),
		other:    pulid.MustNew("org_"),
		actor:    pulid.MustNew("usr_"),
		target:   pulid.MustNew("usr_"),
		held:     grants{},
	}
	f.guard = New(Params{
		Users:       &memberships{orgIDs: []pulid.ID{f.org, f.other}},
		Permissions: &effective{held: f.held},
		Auditor:     f.security,
	})
	f.req = &Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  f.org,
			BuID:   pulid.MustNew("bu_"),
			UserID: f.actor,
		},
		TargetID: f.target,
		Action:   "emailAddress",
	}
	f.held[f.actor] = map[pulid.ID]map[string]permissionports.EffectiveResourcePermission{}
	f.held[f.target] = map[pulid.ID]map[string]permissionports.EffectiveResourcePermission{}

	return f
}

func TestRequireCovers_RefusesALimitedActorOnAnAdministrator(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.held[f.actor][f.org] = userAdmin()
	f.held[f.target][f.org] = everything()

	err := f.guard.RequireCovers(t.Context(), f.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.Len(t, f.security.Changes(), 1)
}

func TestRequireCovers_LooksInEveryOrganizationTheTargetReaches(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.held[f.actor][f.org] = everything()
	f.held[f.target][f.org] = userAdmin()
	f.held[f.target][f.other] = userAdmin()

	err := f.guard.RequireCovers(t.Context(), f.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestRequireCovers_LetsAnActorWhoHoldsEverythingTheTargetHolds(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.held[f.actor][f.org] = everything()
	f.held[f.actor][f.other] = userAdmin()
	f.held[f.target][f.org] = userAdmin()
	f.held[f.target][f.other] = userAdmin()

	require.NoError(t, f.guard.RequireCovers(t.Context(), f.req))
	assert.Empty(t, f.security.Changes())
}

func TestRequireCovers_LeavesAPersonFreeToChangeTheirOwnAccount(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.req.TargetID = f.actor

	require.NoError(t, f.guard.RequireCovers(t.Context(), f.req))
}

func TestRequireCovers_RefusesAnActorWhoIsNotAPerson(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.req.TenantInfo.UserID = pulid.Nil

	err := f.guard.RequireCovers(t.Context(), f.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

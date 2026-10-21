package reportjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingReadScopes struct {
	principal *permissionports.PermissionCheckRequest
	grants    readscope.Grants
}

func (r *recordingReadScopes) Resolver(
	principal *permissionports.PermissionCheckRequest,
) readscope.Resolver {
	r.principal = principal
	return func(context.Context) (readscope.Grants, error) { return r.grants, nil }
}

func TestWithReadScope_NarrowsARunToItsRequestersScope(t *testing.T) {
	t.Parallel()

	team := pulid.MustNew("usr_")
	scopes := &recordingReadScopes{grants: readscope.Grants{
		"worker":   {Scope: readscope.ScopeTeam, Team: []pulid.ID{team}},
		"shipment": {Scope: readscope.ScopeNone},
	}}
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	ctx, err := (&Activities{readScopes: scopes}).withReadScope(t.Context(), tenant)
	require.NoError(t, err)

	encoded, err := readscope.Encode(ctx)
	require.NoError(t, err)
	assert.Equal(t, "shipment:none;worker:team:"+team.String(), encoded)
	require.NotNil(t, scopes.principal)
	assert.Equal(t, permissionports.PrincipalTypeUser, scopes.principal.PrincipalType)
	assert.Equal(t, tenant.UserID, scopes.principal.UserID)
	assert.Equal(t, tenant.UserID, scopes.principal.PrincipalID)
	assert.Equal(t, tenant.OrgID, scopes.principal.OrganizationID)
	assert.Equal(t, tenant.BuID, scopes.principal.BusinessUnitID)
}

func TestWithReadScope_RefusesARunItCannotNarrow(t *testing.T) {
	t.Parallel()

	_, err := (&Activities{}).withReadScope(t.Context(), pagination.TenantInfo{
		UserID: pulid.MustNew("usr_"),
	})
	require.Error(t, err)
}

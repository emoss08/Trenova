package readscopeservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeTeams struct {
	ids   []pulid.ID
	err   error
	calls int
}

func (f *fakeTeams) ManagerIDs(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	worker.ApprovalScope,
	int64,
) ([]pulid.ID, map[pulid.ID]pulid.ID, error) {
	f.calls++
	return f.ids, nil, f.err
}

func enforcing() *config.Config {
	cfg := &config.Config{}
	cfg.Database.RLS.Mode = config.RLSModeEnforce
	return cfg
}

func scopesFor(scopes map[string]permission.DataScope) func(
	*permissionports.BatchPermissionCheckRequest,
) *permissionports.BatchPermissionCheckResult {
	return func(req *permissionports.BatchPermissionCheckRequest) *permissionports.BatchPermissionCheckResult {
		results := make([]permissionports.PermissionCheckResult, len(req.Checks))
		for i, check := range req.Checks {
			scope, ok := scopes[check.Resource]
			results[i] = permissionports.PermissionCheckResult{Allowed: ok, DataScope: scope}
		}
		return &permissionports.BatchPermissionCheckResult{Results: results}
	}
}

func newService(
	t *testing.T,
	cfg *config.Config,
	teams *fakeTeams,
	scopes map[string]permission.DataScope,
) *Service {
	t.Helper()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("CheckBatch", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *permissionports.BatchPermissionCheckRequest) *permissionports.BatchPermissionCheckResult {
			return scopesFor(scopes)(req)
		}, nil).
		Maybe()

	return New(Params{
		Config:   cfg,
		Engine:   engine,
		Teams:    teams,
		Registry: permission.NewRegistry(),
	})
}

func principal() *permissionports.PermissionCheckRequest {
	return &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestGrants_NamesTheNarrowKindsTheDatabaseEnforces(t *testing.T) {
	t.Parallel()

	team := []pulid.ID{pulid.MustNew("usr_")}
	teams := &fakeTeams{ids: team}
	svc := newService(t, enforcing(), teams, map[string]permission.DataScope{
		"shipment": permission.DataScopeOwn,
		"invoice":  permission.DataScopeOrganization,
		"worker":   permission.DataScopeTeam,
	})

	grants, err := svc.Grants(t.Context(), principal())

	require.NoError(t, err)
	assert.Equal(t, readscope.Grants{
		"shipment": {Scope: "own"},
		"worker":   {Scope: "team", Team: team},
	}, grants)
	assert.Equal(t, 1, teams.calls)
}

func TestGrants_IgnoresAScopeTheKindDoesNotDeclare(t *testing.T) {
	t.Parallel()

	svc := newService(t, enforcing(), &fakeTeams{}, map[string]permission.DataScope{
		"shipment": permission.DataScopeTeam,
		"order":    permission.DataScopeOwn,
	})

	grants, err := svc.Grants(t.Context(), principal())

	require.NoError(t, err)
	assert.Equal(t, readscope.Grants{"order": {Scope: "own"}}, grants)
}

func TestGrants_NarrowsNothingWhileTheDatabaseDoesNotEnforceScopes(t *testing.T) {
	t.Parallel()

	svc := newService(t, &config.Config{}, &fakeTeams{}, map[string]permission.DataScope{
		"shipment": permission.DataScopeOwn,
	})

	grants, err := svc.Grants(t.Context(), principal())

	require.NoError(t, err)
	assert.Empty(t, grants)
	assert.False(t, svc.Enforced("shipment", permission.DataScopeOwn))
	assert.True(t, svc.Enforced("capture_batch", permission.DataScopeOwn))
}

func TestGrants_FailsWhenTheTeamCannotBeResolved(t *testing.T) {
	t.Parallel()

	svc := newService(t, enforcing(), &fakeTeams{err: errors.New("database unavailable")},
		map[string]permission.DataScope{"worker": permission.DataScopeTeam})

	_, err := svc.Grants(t.Context(), principal())

	require.Error(t, err)
}

func TestEnforceableScopes_OffersOnlyWhatIsEnforced(t *testing.T) {
	t.Parallel()

	svc := newService(t, enforcing(), &fakeTeams{}, nil)

	assert.Equal(
		t,
		[]permission.DataScope{permission.DataScopeOwn, permission.DataScopeOrganization},
		svc.EnforceableScopes("shipment"),
	)
	assert.Equal(
		t,
		[]permission.DataScope{permission.DataScopeTeam, permission.DataScopeOrganization},
		svc.EnforceableScopes("worker"),
	)
	assert.Equal(
		t,
		[]permission.DataScope{permission.DataScopeOrganization},
		svc.EnforceableScopes("customer"),
	)
}

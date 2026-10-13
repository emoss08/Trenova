package watchtowerjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type fakeControls struct {
	dispatchcontrolports.DispatchControlRepository

	tenants []pagination.TenantInfo
	err     error
}

func (f *fakeControls) ListCheckCallMonitoringTenants(
	context.Context,
) ([]pagination.TenantInfo, error) {
	return f.tenants, f.err
}

type kindCall struct {
	tenant pagination.TenantInfo
	kind   watchtower.SourceKind
	scoped bool
}

type fakeWatchtower struct {
	watchtowerports.WatchtowerService

	calls   []kindCall
	failOrg pulid.ID
}

func (f *fakeWatchtower) ReconcileKind(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind watchtower.SourceKind,
) (*watchtowerports.WatchtowerSweepResult, error) {
	_, scoped := dbscope.TenantFrom(ctx)
	f.calls = append(f.calls, kindCall{tenant: tenant, kind: kind, scoped: scoped})
	if tenant.OrgID == f.failOrg {
		return nil, assert.AnError
	}

	return &watchtowerports.WatchtowerSweepResult{Upserted: 2, Resolved: 1}, nil
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func runNoCheckCallSweep(t *testing.T, a *Activities) (*WatchtowerSweepResult, error) {
	t.Helper()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.NoCheckCallSweepActivity)
	if err != nil {
		return nil, err
	}
	result := new(WatchtowerSweepResult)
	require.NoError(t, value.Get(result))

	return result, nil
}

func TestNoCheckCallSweepReconcilesEachMonitoringTenantInItsScope(t *testing.T) {
	t.Parallel()

	healthy, broken := tenantInfo(), tenantInfo()
	tower := &fakeWatchtower{failOrg: broken.OrgID}
	a := &Activities{
		watchtower: tower,
		controls:   &fakeControls{tenants: []pagination.TenantInfo{healthy, broken}},
		l:          zap.NewNop(),
	}

	result, err := runNoCheckCallSweep(t, a)

	require.NoError(t, err)
	assert.Equal(t, 1, result.OrganizationsProcessed)
	assert.Equal(t, 2, result.Upserted)
	assert.Equal(t, 1, result.Resolved)
	assert.Equal(t, []string{broken.OrgID.String()}, result.FailedOrganizations)
	require.Len(t, tower.calls, 2)
	for _, call := range tower.calls {
		assert.Equal(t, watchtower.SourceNoCheckCall, call.kind)
		assert.True(t, call.scoped, "each tenant runs under its own scope")
	}
}

func TestNoCheckCallSweepFailsWhenTenantsCannotBeListed(t *testing.T) {
	t.Parallel()

	a := &Activities{
		watchtower: &fakeWatchtower{},
		controls:   &fakeControls{err: assert.AnError},
		l:          zap.NewNop(),
	}

	_, err := runNoCheckCallSweep(t, a)

	require.Error(t, err)
}

package watchtowerjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
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

type fakeAttentionRepo struct {
	watchtowerports.WatchtowerRepository

	tenants []pagination.TenantInfo
	err     error
}

func (f *fakeAttentionRepo) ListTenantsNeedingAttention(
	context.Context,
	int64,
) ([]pagination.TenantInfo, error) {
	return f.tenants, f.err
}

type fakeSweeper struct {
	calls   []kindCall
	failOrg pulid.ID
}

func (f *fakeSweeper) SweepAttention(
	ctx context.Context,
	tenant pagination.TenantInfo,
	_ int64,
) (*watchtowerports.ExceptionSweepResult, error) {
	_, scoped := dbscope.TenantFrom(ctx)
	f.calls = append(f.calls, kindCall{tenant: tenant, scoped: scoped})
	if tenant.OrgID == f.failOrg {
		return nil, assert.AnError
	}

	return &watchtowerports.ExceptionSweepResult{Woke: 1, Escalated: 2}, nil
}

func runExceptionAttention(t *testing.T, a *Activities) (*ExceptionAttentionResult, error) {
	t.Helper()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.ExceptionAttentionActivity)
	if err != nil {
		return nil, err
	}
	result := new(ExceptionAttentionResult)
	require.NoError(t, value.Get(result))

	return result, nil
}

func TestExceptionAttentionSweepsEachTenantInItsScopeAndSurvivesOneFailing(t *testing.T) {
	t.Parallel()

	ok, broken := tenantInfo(), tenantInfo()
	sweeper := &fakeSweeper{failOrg: broken.OrgID}
	a := NewActivities(ActivitiesParams{
		Exceptions: sweeper,
		Repo:       &fakeAttentionRepo{tenants: []pagination.TenantInfo{ok, broken}},
		Logger:     zap.NewNop(),
	})

	result, err := runExceptionAttention(t, a)
	require.NoError(t, err)
	assert.Equal(t, 1, result.OrganizationsProcessed)
	assert.Equal(t, 1, result.Woke)
	assert.Equal(t, 2, result.Escalated)
	assert.Equal(t, []string{broken.OrgID.String()}, result.FailedOrganizations)
	require.Len(t, sweeper.calls, 2)
	for _, call := range sweeper.calls {
		assert.True(t, call.scoped)
	}
}

func TestExceptionAttentionFailsWhenTenantsCannotBeListed(t *testing.T) {
	t.Parallel()

	a := NewActivities(ActivitiesParams{
		Exceptions: &fakeSweeper{},
		Repo:       &fakeAttentionRepo{err: assert.AnError},
		Logger:     zap.NewNop(),
	})

	_, err := runExceptionAttention(t, a)
	require.Error(t, err)
}

type fakeTenantSync struct {
	tenantports.TenantSyncRepository

	organizations []tenant.SyncOrganization
	err           error
}

func (f *fakeTenantSync) ListOrganizations(context.Context) ([]tenant.SyncOrganization, error) {
	return f.organizations, f.err
}

func runFactoringSweep(t *testing.T, a *Activities) (*WatchtowerSweepResult, error) {
	t.Helper()

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.FactoringSweepActivity)
	if err != nil {
		return nil, err
	}
	result := new(WatchtowerSweepResult)
	require.NoError(t, value.Get(result))

	return result, nil
}

func TestFactoringSweepReconcilesTheFactoringKindForEachOrganization(t *testing.T) {
	t.Parallel()

	healthy, broken := tenantInfo(), tenantInfo()
	tower := &fakeWatchtower{failOrg: broken.OrgID}
	a := &Activities{
		watchtower: tower,
		tenants: &fakeTenantSync{organizations: []tenant.SyncOrganization{
			{ID: healthy.OrgID, BusinessUnitID: healthy.BuID},
			{ID: broken.OrgID, BusinessUnitID: broken.BuID},
		}},
		l: zap.NewNop(),
	}

	result, err := runFactoringSweep(t, a)

	require.NoError(t, err)
	assert.Equal(t, 1, result.OrganizationsProcessed)
	assert.Equal(t, 2, result.Upserted)
	assert.Equal(t, 1, result.Resolved)
	assert.Equal(t, []string{broken.OrgID.String()}, result.FailedOrganizations)
	require.Len(t, tower.calls, 2)
	for _, call := range tower.calls {
		assert.Equal(t, watchtower.SourceFactoring, call.kind)
		assert.True(t, call.scoped, "each organization runs under its own scope")
	}
	assert.Equal(t, healthy, tower.calls[0].tenant)
}

func TestFactoringSweepFailsWhenOrganizationsCannotBeListed(t *testing.T) {
	t.Parallel()

	a := &Activities{
		watchtower: &fakeWatchtower{},
		tenants:    &fakeTenantSync{err: assert.AnError},
		l:          zap.NewNop(),
	}

	_, err := runFactoringSweep(t, a)

	require.Error(t, err)
}

func TestFactoringSweepIsScheduled(t *testing.T) {
	t.Parallel()

	var found bool
	for _, s := range NewScheduleProvider().GetSchedules() {
		if s.ID == "factoring-watchtower-sweep" {
			found = true
		}
	}
	assert.True(t, found)

	var registered bool
	for _, wf := range RegisterWorkflows() {
		if wf.Name == FactoringSweepWorkflowName {
			registered = true
		}
	}
	assert.True(t, registered)
}

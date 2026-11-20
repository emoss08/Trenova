package tenantjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func payload() *tenantjobscontract.TenantPayload {
	return &tenantjobscontract.TenantPayload{
		InstanceID:     "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10",
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Revision:       9,
	}
}

type removalRig struct {
	suite testsuite.WorkflowTestSuite
	env   *testsuite.TestWorkflowEnvironment
	a     *Activities
	order []string
}

func newRemovalRig(t *testing.T) *removalRig {
	t.Helper()
	r := &removalRig{a: &Activities{}}
	r.env = r.suite.NewTestWorkflowEnvironment()
	r.env.RegisterActivity(r.a)
	return r
}

func (r *removalRig) step(name string) func(mock.Arguments) {
	return func(mock.Arguments) { r.order = append(r.order, name) }
}

func (r *removalRig) happyPath(rounds []*cloudtenantports.PurgeRowsResult) {
	r.env.OnActivity(r.a.CheckRemovalActivity, mock.Anything, mock.Anything).
		Return(true, nil).Run(r.step("check"))
	r.env.OnActivity(r.a.ReleaseMembersActivity, mock.Anything, mock.Anything).
		Return(&tenantlifecycle.RevocationResult{Members: 2, Sessions: 3}, nil).Run(r.step("members"))
	for _, round := range rounds {
		r.env.OnActivity(r.a.PurgeRowsActivity, mock.Anything, mock.Anything).
			Return(round, nil).Once().Run(r.step("rows"))
	}
	r.env.OnActivity(r.a.DeleteOrganizationActivity, mock.Anything, mock.Anything).
		Return(&cloudtenantports.DeleteResult{Deleted: true}, nil).Run(r.step("organization"))
	r.env.OnActivity(r.a.DeleteBusinessUnitActivity, mock.Anything, mock.Anything).
		Return(&cloudtenantports.DeleteResult{Deleted: true}, nil).Run(r.step("unit"))
	r.env.OnActivity(r.a.PurgeStoresActivity, mock.Anything, mock.Anything).
		Return(&StoresResult{Objects: 12, SearchIndexes: 3, CacheKeys: 4}, nil).Run(r.step("stores"))
}

func TestRemovalDeletesInOrderAndReportsTheTenantRemoved(t *testing.T) {
	t.Parallel()
	r := newRemovalRig(t)
	r.happyPath([]*cloudtenantports.PurgeRowsResult{
		{Deleted: 500},
		{Deleted: 120, Left: []string{"users"}},
		{Deleted: 0, Left: []string{"users"}},
		{Deleted: 40, Complete: true},
	})
	r.env.OnActivity(r.a.VerifyRemovedActivity, mock.Anything, mock.Anything).
		Return([]cloudtenantports.TableCount{}, nil).Run(r.step("verify"))
	var finished *FinishInput
	r.env.OnActivity(r.a.FinishRemovalActivity, mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		finished = args[1].(*FinishInput)
		r.order = append(r.order, "finish")
	})

	r.env.ExecuteWorkflow(CloudTenantRemovalWorkflow, payload())

	require.True(t, r.env.IsWorkflowCompleted())
	require.NoError(t, r.env.GetWorkflowError())
	assert.Equal(t, []string{
		"check", "members", "rows", "rows", "rows", "organization", "rows", "unit", "stores", "verify", "finish",
	}, r.order)
	require.NotNil(t, finished)
	assert.Equal(t, int64(660), finished.Rows)
	assert.Equal(t, int64(12), finished.Stores.Objects)
	assert.Equal(t, 3, finished.Sessions)
}

func TestRemovalIsNotReportedWhileRowsRemain(t *testing.T) {
	t.Parallel()
	r := newRemovalRig(t)
	r.happyPath([]*cloudtenantports.PurgeRowsResult{{Complete: true}, {Complete: true}})
	r.env.OnActivity(r.a.VerifyRemovedActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("rows remain", errorRowsRemain, nil))

	r.env.ExecuteWorkflow(CloudTenantRemovalWorkflow, payload())

	require.True(t, r.env.IsWorkflowCompleted())
	require.Error(t, r.env.GetWorkflowError())
	assert.NotContains(t, r.order, "finish")
}

func TestARemovalNoLongerAskedForStops(t *testing.T) {
	t.Parallel()
	r := newRemovalRig(t)
	r.env.OnActivity(r.a.CheckRemovalActivity, mock.Anything, mock.Anything).Return(false, nil)

	r.env.ExecuteWorkflow(CloudTenantRemovalWorkflow, payload())

	require.NoError(t, r.env.GetWorkflowError())
	var out RemovalResult
	require.NoError(t, r.env.GetWorkflowResult(&out))
	assert.True(t, out.Skipped)
}

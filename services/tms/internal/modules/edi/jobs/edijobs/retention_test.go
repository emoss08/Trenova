package edijobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func newRetentionWorkflowTestEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(PurgeEDIRawPayloadsWorkflow)
	return env
}

func TestPurgeEDIRawPayloadsWorkflow_AggregatesTenantResults(t *testing.T) {
	env := newRetentionWorkflowTestEnv(t)
	var a *Activities

	tenants := []EDIRetentionTenant{
		{
			OrganizationID:           pulid.MustNew("org_"),
			BusinessUnitID:           pulid.MustNew("bu_"),
			InboundFileRetentionDays: 30,
			MessageRetentionDays:     90,
		},
		{
			OrganizationID:       pulid.MustNew("org_"),
			BusinessUnitID:       pulid.MustNew("bu_"),
			MessageRetentionDays: 30,
		},
	}
	env.OnActivity(a.ListEDIRetentionTenantsActivity, mock.Anything).
		Return(tenants, nil).
		Once()
	env.OnActivity(a.PurgeEDIRawPayloadsTenantActivity, mock.Anything, tenants[0]).
		Return(&PurgeEDIRawPayloadsTenantResult{InboundFilesPurged: 12, MessagesPurged: 40}, nil).
		Once()
	env.OnActivity(a.PurgeEDIRawPayloadsTenantActivity, mock.Anything, tenants[1]).
		Return(&PurgeEDIRawPayloadsTenantResult{MessagesPurged: 5}, nil).
		Once()

	env.ExecuteWorkflow(PurgeEDIRawPayloadsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	result := new(PurgeEDIRawPayloadsResult)
	require.NoError(t, env.GetWorkflowResult(result))
	require.Equal(t, 2, result.TenantsProcessed)
	require.Equal(t, 0, result.FailedTenants)
	require.Equal(t, int64(12), result.InboundFilesPurged)
	require.Equal(t, int64(45), result.MessagesPurged)
	env.AssertExpectations(t)
}

func TestPurgeEDIRawPayloadsWorkflow_ContinuesPastTenantFailures(t *testing.T) {
	env := newRetentionWorkflowTestEnv(t)
	var a *Activities

	tenants := []EDIRetentionTenant{
		{
			OrganizationID:       pulid.MustNew("org_"),
			BusinessUnitID:       pulid.MustNew("bu_"),
			MessageRetentionDays: 30,
		},
		{
			OrganizationID:           pulid.MustNew("org_"),
			BusinessUnitID:           pulid.MustNew("bu_"),
			InboundFileRetentionDays: 30,
		},
	}
	env.OnActivity(a.ListEDIRetentionTenantsActivity, mock.Anything).
		Return(tenants, nil).
		Once()
	env.OnActivity(a.PurgeEDIRawPayloadsTenantActivity, mock.Anything, tenants[0]).
		Return(nil, errors.New("database timeout")).
		Times(3)
	env.OnActivity(a.PurgeEDIRawPayloadsTenantActivity, mock.Anything, tenants[1]).
		Return(&PurgeEDIRawPayloadsTenantResult{InboundFilesPurged: 3}, nil).
		Once()

	env.ExecuteWorkflow(PurgeEDIRawPayloadsWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	result := new(PurgeEDIRawPayloadsResult)
	require.NoError(t, env.GetWorkflowResult(result))
	require.Equal(t, 1, result.TenantsProcessed)
	require.Equal(t, 1, result.FailedTenants)
	require.Equal(t, int64(3), result.InboundFilesPurged)
	env.AssertExpectations(t)
}

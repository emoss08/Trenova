package aiauditjobs

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/testsuite"
)

func functionName(fn any) string {
	full := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()

	return full[strings.LastIndex(full, ".")+1:]
}

func TestWorkflowsAreRegisteredUnderTheNamesTheServiceStarts(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		serviceports.AIAuditVerifyWorkflowName,
		functionName(VerifyAIAuditChainWorkflow),
	)
	assert.Equal(t, serviceports.AIAuditExportWorkflowName, functionName(AIAuditExportWorkflow))

	for _, wf := range append(RegisterWorkflows(), RegisterExportWorkflows()...) {
		assert.Equal(t, wf.Name, functionName(wf.Fn))
		assert.NotEmpty(t, wf.Description)
	}
}

func TestRegistriesShareTheirQueuesWorkers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, temporaltype.AuditTaskQueue, DomainConfig.TaskQueue)
	assert.Equal(t, temporaltype.ReportTaskQueue, ExportDomainConfig.TaskQueue)
	assert.Equal(t, reportjobs.DomainConfig.WorkerConfig, ExportDomainConfig.WorkerConfig,
		"the export worker must match the report worker's options to share its queue")
	for _, wf := range RegisterExportWorkflows() {
		assert.Equal(t, temporaltype.ReportTaskQueue, wf.TaskQueue)
	}
	for _, wf := range RegisterWorkflows() {
		assert.Equal(t, temporaltype.AuditTaskQueue, wf.TaskQueue)
	}
}

func TestSchedules(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider(&config.Config{}).GetSchedules()
	require.Len(t, schedules, 4)

	ids := map[string]bool{}
	for _, s := range schedules {
		require.NoError(t, s.Validate(), s.ID)
		assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, s.OverlapPolicy, s.ID)
		assert.Equal(t, temporaltype.AuditTaskQueue, s.TaskQueue, s.ID)
		assert.False(t, ids[s.ID])
		ids[s.ID] = true
	}
	assert.Equal(t, ProjectorScheduleID, schedules[0].ID)
	assert.Equal(t, (&config.AIAuditProjectorConfig{}).GetInterval(), schedules[0].Spec.Interval,
		"the projector runs every aiAudit.projector.interval")
}

func newEnv() *testsuite.TestWorkflowEnvironment {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(VerifyAIAuditChainWorkflow)
	env.RegisterWorkflow(VerifyAIAuditTenantsWorkflow)
	env.RegisterWorkflow(AIAuditExportWorkflow)

	var a *Activities
	env.RegisterActivity(a.ListAIAuditTenantsActivity)
	env.RegisterActivity(a.VerifyAIAuditTenantActivity)
	var x *ExportActivities
	env.RegisterActivity(x.RunAIAuditExportActivity)

	return env
}

func TestVerifyWorkflow_ChecksEveryTenantAndKeepsGoingPastAFailure(t *testing.T) {
	t.Parallel()

	env := newEnv()
	tenants := []pagination.TenantInfo{
		{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")},
		{OrgID: pulid.ID("org_b"), BuID: pulid.ID("bu_b")},
	}
	env.OnActivity("ListAIAuditTenantsActivity", mock.Anything).Return(tenants, nil)
	env.OnActivity("VerifyAIAuditTenantActivity", mock.Anything, &tenants[0]).
		Return(&TenantVerification{OrganizationID: "org_a", Status: "Verified"}, nil)
	env.OnActivity("VerifyAIAuditTenantActivity", mock.Anything, &tenants[1]).
		Return(nil, assert.AnError)

	env.ExecuteWorkflow(VerifyAIAuditChainWorkflow, &serviceports.AIAuditVerifyPayload{})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result VerifyResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Len(t, result.Tenants, 1)
	assert.Equal(t, 1, result.Errors)
}

func TestVerifyWorkflow_OneTenantOnDemand(t *testing.T) {
	t.Parallel()

	env := newEnv()
	tenant := pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
	env.OnActivity("VerifyAIAuditTenantActivity", mock.Anything, &tenant).
		Return(&TenantVerification{OrganizationID: "org_a", Status: "Mismatch"}, nil)

	env.ExecuteWorkflow(VerifyAIAuditChainWorkflow, &serviceports.AIAuditVerifyPayload{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})

	require.NoError(t, env.GetWorkflowError())
	var result VerifyResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Len(t, result.Tenants, 1)
	assert.EqualValues(t, "Mismatch", result.Tenants[0].Status)
}

func TestExportWorkflow_RunsTheExport(t *testing.T) {
	t.Parallel()

	env := newEnv()
	payload := &serviceports.AIAuditExportPayload{
		ExportID:       pulid.ID("aiax_1"),
		OrganizationID: pulid.ID("org_a"),
		BusinessUnitID: pulid.ID("bu_a"),
	}
	env.OnActivity("RunAIAuditExportActivity", mock.Anything, payload).
		Return(&ExportResult{ExportID: "aiax_1", Status: "Succeeded", RowCount: 10}, nil)

	env.ExecuteWorkflow(AIAuditExportWorkflow, payload)

	require.NoError(t, env.GetWorkflowError())
	var result ExportResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, int64(10), result.RowCount)
}

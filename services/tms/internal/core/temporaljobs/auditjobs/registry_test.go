package auditjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainConfig(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "audit-worker", DomainConfig.Name)
	assert.Equal(t, temporaltype.AuditTaskQueue, DomainConfig.TaskQueue)
	assert.Equal(t, 10, DomainConfig.WorkerConfig.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 10, DomainConfig.WorkerConfig.MaxConcurrentWorkflowTaskExecutionSize)
	assert.True(t, DomainConfig.WorkerConfig.EnableSessionWorker)
	assert.Equal(t, 30*time.Second, DomainConfig.WorkerConfig.WorkerStopTimeout)
}

func TestConvertWorkflows(t *testing.T) {
	t.Parallel()

	input := []temporaltype.WorkflowDefinition{
		{
			Name:        "Workflow1",
			Fn:          func() {},
			TaskQueue:   "queue1",
			Description: "First workflow",
		},
		{
			Name:        "Workflow2",
			Fn:          func() {},
			TaskQueue:   "queue2",
			Description: "Second workflow",
		},
	}

	result := convertWorkflows(input)

	require.Len(t, result, 2)
	assert.Equal(t, "Workflow1", result[0].Name)
	assert.Equal(t, "First workflow", result[0].Description)
	assert.NotNil(t, result[0].Fn)
	assert.Equal(t, "Workflow2", result[1].Name)
	assert.Equal(t, "Second workflow", result[1].Description)
	assert.NotNil(t, result[1].Fn)
}

func TestConvertWorkflows_Empty(t *testing.T) {
	t.Parallel()

	result := convertWorkflows([]temporaltype.WorkflowDefinition{})

	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestConvertWorkflows_Nil(t *testing.T) {
	t.Parallel()

	result := convertWorkflows(nil)

	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestConvertWorkflows_PreservesOrder(t *testing.T) {
	t.Parallel()

	input := []temporaltype.WorkflowDefinition{
		{Name: "A", Fn: func() {}, Description: "descA"},
		{Name: "B", Fn: func() {}, Description: "descB"},
		{Name: "C", Fn: func() {}, Description: "descC"},
	}

	result := convertWorkflows(input)

	require.Len(t, result, 3)
	assert.Equal(t, "A", result[0].Name)
	assert.Equal(t, "B", result[1].Name)
	assert.Equal(t, "C", result[2].Name)
}

func TestWorkflows_ContainsExpected(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, Workflows)

	names := make([]string, len(Workflows))
	for i, wf := range Workflows {
		names[i] = wf.Name
	}

	assert.Contains(t, names, "ProcessAuditBatchWorkflow")
	assert.Contains(t, names, "ScheduledAuditFlushWorkflow")
	assert.Contains(t, names, "DLQRetryWorkflow")
}

func TestWorkflows_AllHaveFunctions(t *testing.T) {
	t.Parallel()

	for _, wf := range Workflows {
		t.Run(wf.Name, func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, wf.Fn)
			assert.NotEmpty(t, wf.Name)
			assert.NotEmpty(t, wf.Description)
		})
	}
}

func TestRegisterWorkflows(t *testing.T) {
	t.Parallel()

	wfs := RegisterWorkflows()

	require.Len(t, wfs, 3)

	assert.Equal(t, "ProcessAuditBatchWorkflow", wfs[0].Name)
	assert.Equal(t, temporaltype.AuditTaskQueue, wfs[0].TaskQueue)
	assert.NotNil(t, wfs[0].Fn)

	assert.Equal(t, "ScheduledAuditFlushWorkflow", wfs[1].Name)
	assert.Equal(t, temporaltype.AuditTaskQueue, wfs[1].TaskQueue)
	assert.NotNil(t, wfs[1].Fn)

	assert.Equal(t, "DLQRetryWorkflow", wfs[2].Name)
	assert.Equal(t, temporaltype.AuditTaskQueue, wfs[2].TaskQueue)
	assert.NotNil(t, wfs[2].Fn)
}

func TestRegisterWorkflows_Descriptions(t *testing.T) {
	t.Parallel()

	wfs := RegisterWorkflows()

	for _, wf := range wfs {
		t.Run(wf.Name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, wf.Description)
		})
	}
}

func TestDomainConfig_MatchesDefaultWorkerConfig(t *testing.T) {
	t.Parallel()

	defaultCfg := registry.DefaultWorkerConfig()

	assert.Equal(
		t,
		defaultCfg.MaxConcurrentActivityExecutionSize,
		DomainConfig.WorkerConfig.MaxConcurrentActivityExecutionSize,
	)
	assert.Equal(
		t,
		defaultCfg.MaxConcurrentWorkflowTaskExecutionSize,
		DomainConfig.WorkerConfig.MaxConcurrentWorkflowTaskExecutionSize,
	)
	assert.Equal(
		t,
		defaultCfg.MaxConcurrentWorkflowTaskPollers,
		DomainConfig.WorkerConfig.MaxConcurrentWorkflowTaskPollers,
	)
	assert.Equal(
		t,
		defaultCfg.MaxConcurrentActivityTaskPollers,
		DomainConfig.WorkerConfig.MaxConcurrentActivityTaskPollers,
	)
	assert.Equal(t, defaultCfg.EnableSessionWorker, DomainConfig.WorkerConfig.EnableSessionWorker)
	assert.Equal(t, defaultCfg.WorkerStopTimeout, DomainConfig.WorkerConfig.WorkerStopTimeout)
}

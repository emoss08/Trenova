package registry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewWorkerManager(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	require.NotNil(t, wm)
	assert.NotNil(t, wm.workers)
	assert.NotNil(t, wm.registries)
	assert.Empty(t, wm.workers)
	assert.Empty(t, wm.registries)
}

func TestWorkerManager_GetWorker_NotFound(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	w, exists := wm.GetWorker("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, w)
}

func TestWorkerManager_GetWorker_EmptyName(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	w, exists := wm.GetWorker("")
	assert.False(t, exists)
	assert.Nil(t, w)
}

func TestWorkerConfig_DefaultValues(t *testing.T) {
	t.Parallel()

	cfg := DefaultWorkerConfig()

	opts := cfg.ToWorkerOptions()
	assert.Equal(t, 10, opts.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 10, opts.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 2, opts.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 2, opts.MaxConcurrentActivityTaskPollers)
	assert.True(t, opts.EnableSessionWorker)
	assert.Equal(t, 30*time.Second, opts.WorkerStopTimeout)
}

func TestWorkerConfig_ZeroValues(t *testing.T) {
	t.Parallel()

	cfg := WorkerConfig{}
	opts := cfg.ToWorkerOptions()

	assert.Equal(t, 0, opts.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 0, opts.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 0, opts.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 0, opts.MaxConcurrentActivityTaskPollers)
	assert.False(t, opts.EnableSessionWorker)
	assert.Equal(t, time.Duration(0), opts.WorkerStopTimeout)
}

func TestWorkerConfig_CustomValues(t *testing.T) {
	t.Parallel()

	cfg := WorkerConfig{
		MaxConcurrentActivityExecutionSize:     50,
		MaxConcurrentWorkflowTaskExecutionSize: 25,
		MaxConcurrentWorkflowTaskPollers:       8,
		MaxConcurrentActivityTaskPollers:       12,
		EnableSessionWorker:                    false,
		WorkerStopTimeout:                      2 * time.Minute,
	}

	opts := cfg.ToWorkerOptions()
	assert.Equal(t, 50, opts.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 25, opts.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 8, opts.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 12, opts.MaxConcurrentActivityTaskPollers)
	assert.False(t, opts.EnableSessionWorker)
	assert.Equal(t, 2*time.Minute, opts.WorkerStopTimeout)
}

func TestDomainConfig_Fields(t *testing.T) {
	t.Parallel()

	cfg := DomainConfig{
		Name:         "test-worker",
		TaskQueue:    "test-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}

	assert.Equal(t, "test-worker", cfg.Name)
	assert.Equal(t, "test-queue", cfg.TaskQueue)
	assert.Equal(t, 10, cfg.WorkerConfig.MaxConcurrentActivityExecutionSize)
}

func TestWorkflowDefinition_Fields(t *testing.T) {
	t.Parallel()

	fn := func() {}
	wf := WorkflowDefinition{
		Name:        "TestWorkflow",
		Fn:          fn,
		Description: "A test workflow",
	}

	assert.Equal(t, "TestWorkflow", wf.Name)
	assert.NotNil(t, wf.Fn)
	assert.Equal(t, "A test workflow", wf.Description)
}

func TestDomainRegistry_GetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   DomainConfig
		expected string
	}{
		{
			name:     "simple name",
			config:   DomainConfig{Name: "audit-worker"},
			expected: "audit-worker",
		},
		{
			name:     "empty name",
			config:   DomainConfig{Name: ""},
			expected: "",
		},
		{
			name:     "name with special chars",
			config:   DomainConfig{Name: "worker-123_test"},
			expected: "worker-123_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := NewDomainRegistry(&tt.config, nil, nil, zap.NewNop())
			assert.Equal(t, tt.expected, reg.GetName())
		})
	}
}

func TestDomainRegistry_GetTaskQueue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   DomainConfig
		expected string
	}{
		{
			name:     "simple queue",
			config:   DomainConfig{TaskQueue: "audit-task-queue"},
			expected: "audit-task-queue",
		},
		{
			name:     "empty queue",
			config:   DomainConfig{TaskQueue: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := NewDomainRegistry(&tt.config, nil, nil, zap.NewNop())
			assert.Equal(t, tt.expected, reg.GetTaskQueue())
		})
	}
}

func TestDomainRegistry_GetWorkerOptions_CustomConfig(t *testing.T) {
	t.Parallel()

	config := DomainConfig{
		Name:      "custom",
		TaskQueue: "custom-queue",
		WorkerConfig: WorkerConfig{
			MaxConcurrentActivityExecutionSize:     100,
			MaxConcurrentWorkflowTaskExecutionSize: 50,
			MaxConcurrentWorkflowTaskPollers:       5,
			MaxConcurrentActivityTaskPollers:       10,
			EnableSessionWorker:                    false,
			WorkerStopTimeout:                      45 * time.Second,
		},
	}

	reg := NewDomainRegistry(&config, nil, nil, zap.NewNop())
	opts := reg.GetWorkerOptions()

	assert.Equal(t, 100, opts.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 50, opts.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 5, opts.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 10, opts.MaxConcurrentActivityTaskPollers)
	assert.False(t, opts.EnableSessionWorker)
	assert.Equal(t, 45*time.Second, opts.WorkerStopTimeout)
}

func TestDomainRegistry_WithActivitiesAndWorkflows(t *testing.T) {
	t.Parallel()

	type testActivities struct{}
	activities := &testActivities{}
	workflows := []WorkflowDefinition{
		{Name: "Workflow1", Fn: func() {}, Description: "First workflow"},
		{Name: "Workflow2", Fn: func() {}, Description: "Second workflow"},
	}

	config := DomainConfig{
		Name:         "multi-workflow",
		TaskQueue:    "multi-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}

	reg := NewDomainRegistry(&config, activities, workflows, zap.NewNop())

	require.NotNil(t, reg)
	assert.Equal(t, "multi-workflow", reg.GetName())
	assert.Equal(t, "multi-queue", reg.GetTaskQueue())
	assert.Equal(t, activities, reg.activities)
	assert.Len(t, reg.workflows, 2)
}

func TestDomainRegistry_NilWorkflows(t *testing.T) {
	t.Parallel()

	config := DomainConfig{
		Name:         "nil-workflows",
		TaskQueue:    "nil-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}

	reg := NewDomainRegistry(&config, nil, nil, zap.NewNop())

	require.NotNil(t, reg)
	assert.Nil(t, reg.workflows)
	assert.Nil(t, reg.activities)
}

func TestDomainRegistry_EmptyWorkflows(t *testing.T) {
	t.Parallel()

	config := DomainConfig{
		Name:         "empty-workflows",
		TaskQueue:    "empty-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}

	reg := NewDomainRegistry(&config, nil, []WorkflowDefinition{}, zap.NewNop())

	require.NotNil(t, reg)
	assert.Empty(t, reg.workflows)
}

func TestErrNoWorkersRegistered(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, ErrNoWorkersRegistered)
	assert.Equal(t, "no workers registered", ErrNoWorkersRegistered.Error())
}

func TestErrTemporalClientNotConfigured(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, ErrTemporalClientNotConfigured)
	assert.Equal(t, "temporal client is not configured", ErrTemporalClientNotConfigured.Error())
}

func TestWorkerManager_StartAll_NoWorkers(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	err := wm.StartAll(context.TODO())
	assert.ErrorIs(t, err, ErrNoWorkersRegistered)
}

func TestWorkerManager_StopAll_NoWorkers(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	err := wm.StopAll(context.TODO())
	assert.NoError(t, err)
}

func TestWorkerManager_Register_NilClient(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	wm := NewWorkerManager(nil, logger)

	config := DomainConfig{
		Name:         "test-reg",
		TaskQueue:    "test-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}
	reg := NewDomainRegistry(&config, nil, nil, logger)

	err := wm.Register(reg)
	assert.ErrorIs(t, err, ErrTemporalClientNotConfigured)
	assert.ErrorContains(t, err, "worker=test-reg")
}

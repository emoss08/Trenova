package registry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
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
	assert.False(t, opts.EnableSessionWorker)
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

func TestWorkerManager_Register_SharedTaskQueueUsesOneWorker(t *testing.T) {
	t.Parallel()

	c, err := client.NewLazyClient(client.Options{})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	wm := NewWorkerManager(c, zap.NewNop())
	first := newTestWorkerRegistry("first-worker", "shared-queue", DefaultWorkerConfig())
	second := newTestWorkerRegistry("second-worker", "shared-queue", DefaultWorkerConfig())

	require.NoError(t, wm.Register(first))
	require.NoError(t, wm.Register(second))

	firstWorker, exists := wm.GetWorker("first-worker")
	require.True(t, exists)
	secondWorker, exists := wm.GetWorker("second-worker")
	require.True(t, exists)

	assert.Equal(t, firstWorker, secondWorker)
	assert.Equal(t, first.registeredWorker, second.registeredWorker)
	assert.Len(t, wm.queueWorkers, 1)
	assert.Len(t, wm.workers, 2)
	assert.Equal(t, 1, first.activityRegistrations)
	assert.Equal(t, 1, first.workflowRegistrations)
	assert.Equal(t, 1, second.activityRegistrations)
	assert.Equal(t, 1, second.workflowRegistrations)
}

func TestWorkerManager_StartAllStopAll_UsesUniqueQueueWorkers(t *testing.T) {
	t.Parallel()

	sharedWorker := &countingWorker{}
	otherWorker := &countingWorker{}
	wm := NewWorkerManager(nil, zap.NewNop())
	wm.workers["first-worker"] = sharedWorker
	wm.workers["second-worker"] = sharedWorker
	wm.workers["third-worker"] = otherWorker
	wm.queueWorkers["shared-queue"] = sharedWorker
	wm.queueWorkers["other-queue"] = otherWorker

	require.NoError(t, wm.StartAll(context.TODO()))
	require.NoError(t, wm.StopAll(context.TODO()))

	assert.Equal(t, 1, sharedWorker.startCount)
	assert.Equal(t, 1, sharedWorker.stopCount)
	assert.Equal(t, 1, otherWorker.startCount)
	assert.Equal(t, 1, otherWorker.stopCount)
}

func TestWorkerManager_StartAll_ReturnsQueueWorkerErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("start failed")
	failingWorker := &countingWorker{startErr: expectedErr}
	wm := NewWorkerManager(nil, zap.NewNop())
	wm.workers["first-worker"] = failingWorker
	wm.queueWorkers["shared-queue"] = failingWorker

	err := wm.StartAll(context.TODO())

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to start 1 worker(s)")
	assert.ErrorContains(t, err, "shared-queue")
	assert.Equal(t, 1, failingWorker.startCount)
}

func TestWorkerManager_Register_DuplicateRegistryNameFails(t *testing.T) {
	t.Parallel()

	c, err := client.NewLazyClient(client.Options{})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	wm := NewWorkerManager(c, zap.NewNop())
	first := newTestWorkerRegistry("duplicate-worker", "first-queue", DefaultWorkerConfig())
	second := newTestWorkerRegistry("duplicate-worker", "second-queue", DefaultWorkerConfig())

	require.NoError(t, wm.Register(first))

	err = wm.Register(second)

	require.Error(t, err)
	assert.ErrorContains(t, err, "worker duplicate-worker already registered")
	assert.Len(t, wm.queueWorkers, 1)
	assert.Len(t, wm.workers, 1)
}

func TestWorkerManager_Register_IncompatibleSharedQueueOptionsFail(t *testing.T) {
	t.Parallel()

	c, err := client.NewLazyClient(client.Options{})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	wm := NewWorkerManager(c, zap.NewNop())
	firstConfig := DefaultWorkerConfig()
	secondConfig := DefaultWorkerConfig()
	secondConfig.MaxConcurrentActivityExecutionSize++
	first := newTestWorkerRegistry("first-worker", "shared-queue", firstConfig)
	second := newTestWorkerRegistry("second-worker", "shared-queue", secondConfig)

	require.NoError(t, wm.Register(first))

	err = wm.Register(second)

	require.Error(t, err)
	assert.ErrorContains(t, err, `worker options for task queue "shared-queue" are incompatible`)
	assert.Len(t, wm.queueWorkers, 1)
	assert.Len(t, wm.workers, 1)
	_, exists := wm.GetWorker("second-worker")
	assert.False(t, exists)
}

type testWorkerRegistry struct {
	name                  string
	taskQueue             string
	options               worker.Options
	registeredWorker      worker.Worker
	activityRegistrations int
	workflowRegistrations int
}

func newTestWorkerRegistry(name, taskQueue string, cfg WorkerConfig) *testWorkerRegistry {
	return &testWorkerRegistry{
		name:      name,
		taskQueue: taskQueue,
		options:   cfg.ToWorkerOptions(),
	}
}

func (r *testWorkerRegistry) GetName() string {
	return r.name
}

func (r *testWorkerRegistry) GetTaskQueue() string {
	return r.taskQueue
}

func (r *testWorkerRegistry) RegisterActivities(w worker.Worker) error {
	r.registeredWorker = w
	r.activityRegistrations++
	return nil
}

func (r *testWorkerRegistry) RegisterWorkflows(w worker.Worker) error {
	r.registeredWorker = w
	r.workflowRegistrations++
	return nil
}

func (r *testWorkerRegistry) GetWorkerOptions() worker.Options {
	return r.options
}

type countingWorker struct {
	startCount int
	stopCount  int
	startErr   error
}

func (w *countingWorker) Start() error {
	w.startCount++
	return w.startErr
}

func (w *countingWorker) Run(_ <-chan interface{}) error {
	return w.Start()
}

func (w *countingWorker) Stop() {
	w.stopCount++
}

func (w *countingWorker) RegisterWorkflow(_ interface{}) {}

func (w *countingWorker) RegisterWorkflowWithOptions(
	_ interface{},
	_ workflow.RegisterOptions,
) {
}

func (w *countingWorker) RegisterDynamicWorkflow(
	_ interface{},
	_ workflow.DynamicRegisterOptions,
) {
}

func (w *countingWorker) RegisterActivity(_ interface{}) {}

func (w *countingWorker) RegisterActivityWithOptions(
	_ interface{},
	_ activity.RegisterOptions,
) {
}

func (w *countingWorker) RegisterDynamicActivity(
	_ interface{},
	_ activity.DynamicRegisterOptions,
) {
}

func (w *countingWorker) RegisterNexusService(_ *nexus.Service) {}

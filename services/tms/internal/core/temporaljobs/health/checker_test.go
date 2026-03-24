package health

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
)

type mockTemporalClient struct{}

func (m *mockTemporalClient) Close() {}

type fakeWorkerRegistry struct {
	name      string
	taskQueue string
}

func (f *fakeWorkerRegistry) GetName() string                          { return f.name }
func (f *fakeWorkerRegistry) GetTaskQueue() string                     { return f.taskQueue }
func (f *fakeWorkerRegistry) RegisterActivities(w worker.Worker) error { return nil }
func (f *fakeWorkerRegistry) RegisterWorkflows(w worker.Worker) error  { return nil }
func (f *fakeWorkerRegistry) GetWorkerOptions() worker.Options         { return worker.Options{} }

func TestNewChecker(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	require.NotNil(t, checker)
	assert.NotNil(t, checker.workers)
	assert.NotNil(t, checker.logger)
	assert.Equal(t, manager, checker.workerManager)
}

func TestChecker_RegisterWorker(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("test-worker", "test-queue")

	checker.mu.RLock()
	defer checker.mu.RUnlock()

	info, exists := checker.workers["test-worker"]
	require.True(t, exists)
	assert.Equal(t, "test-worker", info.name)
	assert.Equal(t, "test-queue", info.taskQueue)
	assert.False(t, info.registrationTime.IsZero())
}

func TestChecker_RegisterWorker_Multiple(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("worker-1", "queue-1")
	checker.RegisterWorker("worker-2", "queue-2")
	checker.RegisterWorker("worker-3", "queue-3")

	checker.mu.RLock()
	defer checker.mu.RUnlock()

	assert.Len(t, checker.workers, 3)
}

func TestChecker_RegisterWorker_Overwrite(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("worker-1", "queue-old")
	checker.RegisterWorker("worker-1", "queue-new")

	checker.mu.RLock()
	defer checker.mu.RUnlock()

	assert.Len(t, checker.workers, 1)
	assert.Equal(t, "queue-new", checker.workers["worker-1"].taskQueue)
}

func TestChecker_CheckHealth_NoWorkers(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	statuses := checker.CheckHealth(t.Context())
	assert.Empty(t, statuses)
}

func TestChecker_CheckHealth_WorkerNotInManager(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("missing-worker", "some-queue")

	statuses := checker.CheckHealth(t.Context())
	require.Len(t, statuses, 1)
	assert.Equal(t, "missing-worker", statuses[0].WorkerName)
	assert.Equal(t, "some-queue", statuses[0].TaskQueue)
	assert.False(t, statuses[0].IsHealthy)
	assert.Equal(t, "worker not registered with manager", statuses[0].ErrorMessage)
	assert.False(t, statuses[0].RegistrationTime.IsZero())
}

func TestChecker_CheckHealth_MultipleWorkers_AllMissing(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("worker-a", "queue-a")
	checker.RegisterWorker("worker-b", "queue-b")

	statuses := checker.CheckHealth(t.Context())
	require.Len(t, statuses, 2)

	for _, status := range statuses {
		assert.False(t, status.IsHealthy)
		assert.Equal(t, "worker not registered with manager", status.ErrorMessage)
	}
}

func TestChecker_IsReady_NoWorkers(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	assert.False(t, checker.IsReady())
}

func TestChecker_IsReady_WorkerNotInManager(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("worker-1", "queue-1")

	assert.False(t, checker.IsReady())
}

func TestChecker_IsReady_MultipleWorkers_OneMissing(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("worker-1", "queue-1")
	checker.RegisterWorker("worker-2", "queue-2")

	assert.False(t, checker.IsReady())
}

func TestChecker_ImplementsHealthProbe(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	var _ HealthProbe = checker
}

func TestChecker_ImplementsWorkerRegistrar(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	var _ WorkerRegistrar = checker
}

func TestWorkerHealthStatus_Fields(t *testing.T) {
	t.Parallel()

	manager := registry.NewWorkerManager(nil, zap.NewNop())
	checker := NewChecker(manager, zap.NewNop())

	checker.RegisterWorker("status-test", "status-queue")

	statuses := checker.CheckHealth(t.Context())
	require.Len(t, statuses, 1)

	status := statuses[0]
	assert.Equal(t, "status-test", status.WorkerName)
	assert.Equal(t, "status-queue", status.TaskQueue)
	assert.False(t, status.RegistrationTime.IsZero())
}

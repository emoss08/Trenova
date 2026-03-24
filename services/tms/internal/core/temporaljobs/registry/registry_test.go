package registry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultWorkerConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultWorkerConfig()
	assert.Equal(t, 10, cfg.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 10, cfg.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 2, cfg.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 2, cfg.MaxConcurrentActivityTaskPollers)
	assert.True(t, cfg.EnableSessionWorker)
	assert.Equal(t, 30*time.Second, cfg.WorkerStopTimeout)
}

func TestWorkerConfig_ToWorkerOptions(t *testing.T) {
	t.Parallel()
	cfg := WorkerConfig{
		MaxConcurrentActivityExecutionSize:     5,
		MaxConcurrentWorkflowTaskExecutionSize: 8,
		MaxConcurrentWorkflowTaskPollers:       3,
		MaxConcurrentActivityTaskPollers:       4,
		EnableSessionWorker:                    false,
		WorkerStopTimeout:                      10 * time.Second,
	}

	opts := cfg.ToWorkerOptions()
	assert.Equal(t, 5, opts.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, 8, opts.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, 3, opts.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, 4, opts.MaxConcurrentActivityTaskPollers)
	assert.False(t, opts.EnableSessionWorker)
	assert.Equal(t, 10*time.Second, opts.WorkerStopTimeout)
}

func TestNewDomainRegistry(t *testing.T) {
	t.Parallel()
	config := &DomainConfig{
		Name:         "test-domain",
		TaskQueue:    "test-queue",
		WorkerConfig: DefaultWorkerConfig(),
	}

	reg := NewDomainRegistry(config, nil, nil, zap.NewNop())
	require.NotNil(t, reg)
	assert.Equal(t, "test-domain", reg.GetName())
	assert.Equal(t, "test-queue", reg.GetTaskQueue())
}

func TestDomainRegistry_GetWorkerOptions(t *testing.T) {
	t.Parallel()
	config := &DomainConfig{
		Name:      "test",
		TaskQueue: "queue",
		WorkerConfig: WorkerConfig{
			MaxConcurrentActivityExecutionSize: 20,
			EnableSessionWorker:                true,
			WorkerStopTimeout:                  60 * time.Second,
		},
	}

	reg := NewDomainRegistry(config, nil, nil, zap.NewNop())
	opts := reg.GetWorkerOptions()
	assert.Equal(t, 20, opts.MaxConcurrentActivityExecutionSize)
	assert.True(t, opts.EnableSessionWorker)
	assert.Equal(t, 60*time.Second, opts.WorkerStopTimeout)
}

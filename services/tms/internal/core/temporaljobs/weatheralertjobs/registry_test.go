package weatheralertjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/stretchr/testify/assert"
)

func TestDomainConfig_MatchesDefaultWorkerConfig(t *testing.T) {
	t.Parallel()

	defaultConfig := registry.DefaultWorkerConfig()

	assert.Equal(
		t,
		defaultConfig.MaxConcurrentActivityExecutionSize,
		DomainConfig.WorkerConfig.MaxConcurrentActivityExecutionSize,
	)
	assert.Equal(
		t,
		defaultConfig.MaxConcurrentWorkflowTaskExecutionSize,
		DomainConfig.WorkerConfig.MaxConcurrentWorkflowTaskExecutionSize,
	)
	assert.Equal(
		t,
		defaultConfig.MaxConcurrentWorkflowTaskPollers,
		DomainConfig.WorkerConfig.MaxConcurrentWorkflowTaskPollers,
	)
	assert.Equal(
		t,
		defaultConfig.MaxConcurrentActivityTaskPollers,
		DomainConfig.WorkerConfig.MaxConcurrentActivityTaskPollers,
	)
	assert.Equal(t, defaultConfig.WorkerStopTimeout, DomainConfig.WorkerConfig.WorkerStopTimeout)
	assert.Equal(t, defaultConfig.EnableSessionWorker, DomainConfig.WorkerConfig.EnableSessionWorker)
}

func TestDomainConfig_DisablesSessionWorker(t *testing.T) {
	t.Parallel()

	assert.False(t, DomainConfig.WorkerConfig.EnableSessionWorker)
}

package weatheralertjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/stretchr/testify/assert"
)

func TestWeatherAlertWorkerConfig_DisablesSessionWorker(t *testing.T) {
	t.Parallel()

	defaultConfig := registry.DefaultWorkerConfig()
	config := weatherAlertWorkerConfig()

	assert.Equal(t, defaultConfig.MaxConcurrentActivityExecutionSize, config.MaxConcurrentActivityExecutionSize)
	assert.Equal(t, defaultConfig.MaxConcurrentWorkflowTaskExecutionSize, config.MaxConcurrentWorkflowTaskExecutionSize)
	assert.Equal(t, defaultConfig.MaxConcurrentWorkflowTaskPollers, config.MaxConcurrentWorkflowTaskPollers)
	assert.Equal(t, defaultConfig.MaxConcurrentActivityTaskPollers, config.MaxConcurrentActivityTaskPollers)
	assert.Equal(t, defaultConfig.WorkerStopTimeout, config.WorkerStopTimeout)
	assert.False(t, config.EnableSessionWorker)
}

func TestDomainConfig_DisablesSessionWorker(t *testing.T) {
	t.Parallel()

	assert.False(t, DomainConfig.WorkerConfig.EnableSessionWorker)
}

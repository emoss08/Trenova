package weatheralertjobs

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(
		t,
		defaultConfig.EnableSessionWorker,
		DomainConfig.WorkerConfig.EnableSessionWorker,
	)
}

func TestDomainConfig_DisablesSessionWorker(t *testing.T) {
	t.Parallel()

	assert.False(t, DomainConfig.WorkerConfig.EnableSessionWorker)
}

func TestWorkflows_RegisterThePollWorkflow(t *testing.T) {
	t.Parallel()

	require.Len(t, Workflows, 1)
	assert.Equal(t, PollNWSAlertsWorkflowName, Workflows[0].Name)
	assert.NotNil(t, Workflows[0].Fn)
}

func TestActivities_KeepThePerTenantActivitiesUntilOldRunsDrain(t *testing.T) {
	t.Parallel()

	activities := reflect.TypeFor[*Activities]()
	for _, name := range []string{
		"PollNWSAlertsActivity",
		"ExpireStaleWeatherAlertsActivity",
		"ListWeatherAlertTenantsActivity",
		"PollNWSAlertsForTenantActivity",
	} {
		_, ok := activities.MethodByName(name)
		assert.True(t, ok, name)
	}
}

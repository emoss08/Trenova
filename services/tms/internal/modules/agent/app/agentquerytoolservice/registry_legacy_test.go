package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
An entity's reader is internal to records_query and record_get: the registry
still resolves it by name, for the records tools to call, but never offers it to
a model as a tool of its own. The legacy search_workers alias is gone with it.
*/
func TestRegistry_ResolvesAnEntityReaderButNeverAdvertisesIt(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []agenttoolports.AgentQueryTool{
			workerquerytools.NewListWorkersTool(
				&querytooltest.FakeWorkerRepo{},
				&querytooltest.FakePermissions{},
			),
		},
	})

	tool, ok := reg.Get("worker_list")
	require.True(t, ok)
	assert.Equal(t, "worker_list", tool.Name())

	_, ok = reg.Get("search_workers")
	assert.False(t, ok, "the legacy alias no longer resolves")

	for _, descriptor := range reg.Descriptors() {
		assert.NotEqual(t, "worker_list", descriptor.Name)
		assert.NotEqual(t, "search_workers", descriptor.Name)
	}
}

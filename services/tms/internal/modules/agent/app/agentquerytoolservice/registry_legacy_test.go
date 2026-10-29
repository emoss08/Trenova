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
The tool name is persisted in agent_definitions.tool_names and as a key in
tool_tiers, so a rename is a data change, not a symbol change. The migration
rewrites both columns, but a definition written by an API client that hard-coded
the old name would otherwise lose the tool silently — which is worse than an
error, because the agent simply stops being able to look drivers up.
*/
func TestRegistry_ResolvesTheLegacyWorkerSearchName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []agenttoolports.AgentQueryTool{
			workerquerytools.NewListWorkersTool(
				&querytooltest.FakeWorkerRepo{},
				&querytooltest.FakePermissions{},
			),
		},
	})

	tool, ok := reg.Get("search_workers")
	require.True(t, ok, "an agent saved before the rename still has its tool")
	assert.Equal(t, "worker_list", tool.Name())
}

func TestRegistry_DoesNotAdvertiseTheLegacyName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(RegistryParams{
		Tools: []agenttoolports.AgentQueryTool{
			workerquerytools.NewListWorkersTool(
				&querytooltest.FakeWorkerRepo{},
				&querytooltest.FakePermissions{},
			),
		},
	})

	for _, descriptor := range reg.Descriptors() {
		assert.NotEqual(t, "search_workers", descriptor.Name,
			"the alias is for reading old definitions, not for the model to call")
	}
}

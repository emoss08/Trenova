package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func deferredCoreRuntime(t *testing.T) (*Service, []string) {
	t.Helper()

	service, names := wideRuntime(t)
	core := []agenttoolports.AgentQueryTool{
		describedTool(
			agentdefinition.CoreToolRecallMemory,
			"Look up what the organization recorded.",
		),
		describedTool(agentdefinition.CoreToolWaitUntil,
			"Wait until something happens: a truck arriving, a reply, a time."),
		describedTool(
			agentdefinition.CoreToolCancelWait,
			"Stop waiting on something you were waiting for.",
		),
	}
	query := service.queryTools.(*stubQueryRegistry)
	query.Tools = append(query.Tools, core...)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: service.actionTools,
	})

	return service, append(names, agentdefinition.CoreToolRecallMemory,
		agentdefinition.CoreToolWaitUntil, agentdefinition.CoreToolCancelWait)
}

func TestNewToolSet_ATurnThatCanSearchDefersTheCoreToolsFewTurnsCall(t *testing.T) {
	t.Parallel()

	service, names := deferredCoreRuntime(t)
	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "say hello",
	})

	require.True(t, set.disclosed)
	assert.True(t, set.offers(agentdefinition.CoreToolRecallMemory), "memory stays loaded")
	assert.False(t, set.offers(agentdefinition.CoreToolWaitUntil))
	assert.False(t, set.offers(agentdefinition.CoreToolCancelWait))
	assert.Contains(t, set.deferred, agentdefinition.CoreToolWaitUntil,
		"a core tool not loaded is still declared, so find_tools can load it")

	asked := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "wait until the truck is arriving at the dock",
	})
	assert.True(t, asked.offers(agentdefinition.CoreToolWaitUntil),
		"a request that names what a deferred core tool does opens with it")
}

func TestNewToolSet_ATurnThatCannotSearchLoadsEveryCoreTool(t *testing.T) {
	t.Parallel()

	service, _ := deferredCoreRuntime(t)
	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition("worker_list", agentdefinition.CoreToolWaitUntil),
		actor:      testActor(),
		input:      "say hello",
	})

	require.False(t, set.searchable())
	assert.True(t, set.offers(agentdefinition.CoreToolWaitUntil))
	assert.Empty(t, set.deferred, "with nothing to search, nothing is declared deferred")
}

func TestFindFor_LoadsWhatTheActivityFoundIntoTheTurnAsReferences(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	req := &agentruntimeports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
	}
	turn := service.RestoreTurn(req, service.OpenTurn(t.Context(), req).State())
	require.False(t, turn.tools.offers("equipment_list_trailers"))
	declared := turn.completionRequest()

	found := service.FindFor(t.Context(), req, turn.ToolsState(),
		map[string]any{"need": "trailer inspection date"})
	turn.LoadTools(found.Loaded)

	require.Contains(t, found.Loaded, "equipment_list_trailers")
	assert.True(t, turn.tools.offers("equipment_list_trailers"))
	assert.Equal(t, declared.Tools, turn.completionRequest().Tools,
		"loading a tool leaves what the turn declared as it was")
	assert.Equal(t, declared.DeferredTools, turn.completionRequest().DeferredTools)
	assert.Contains(t, turn.tools.takeLoaded(), "equipment_list_trailers",
		"the next tool result names it")

	restored := service.RestoreTurn(req, turn.State())
	assert.True(
		t,
		restored.tools.offers("equipment_list_trailers"),
		"a restored turn keeps what it loaded",
	)
}

func TestStreamCompletion_DeclaresEachDeferredToolInFullAndSorted(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	req := &agentruntimeports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
	}
	turn := service.OpenTurn(t.Context(), req)
	request := turn.completionRequest()
	require.NotEmpty(t, request.DeferredTools)
	for _, spec := range request.Tools {
		assert.False(t, spec.Deferred, "a turn's own request carries only what is loaded in full")
	}

	declared := service.declareTools(request)

	assert.Empty(t, declared.DeferredTools)
	assert.Len(t, declared.Tools, len(request.Tools)+len(request.DeferredTools))
	deferred := 0
	for idx, spec := range declared.Tools {
		if idx > 0 {
			assert.Less(t, declared.Tools[idx-1].Name, spec.Name, "sorted by name")
		}
		if spec.Deferred {
			deferred++
			assert.NotEmpty(t, spec.Parameters, "a deferred tool is declared with its schema")
		}
	}
	assert.Equal(t, len(request.DeferredTools), deferred)
	assert.Equal(t, declared.Tools, service.declareTools(turn.completionRequest()).Tools,
		"the same declaration every call")
}

func TestToolMap_GroupsTheToolsByAreaOfTrenova(t *testing.T) {
	t.Parallel()

	query := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "shipment_list", Resource: permission.ResourceShipment},
		&agenttest.StubQueryTool{ToolName: "shipment_get", Resource: permission.ResourceShipment},
		&agenttest.StubQueryTool{ToolName: "invoice_list", Resource: permission.ResourceInvoice},
	}}
	service := &Service{
		logger:      zap.NewNop(),
		queryTools:  query,
		actionTools: &stubActionRegistry{},
		registry:    permission.NewRegistry(),
	}

	areas := service.toolMap([]string{
		"shipment_list", "shipment_get", "invoice_list", "shipment_list", "find_tools",
	})

	require.Len(t, areas, 2, "a runtime tool has no area; a name given twice counts once")
	assert.Equal(t, "Billing", areas[0].Name)
	assert.Equal(t, 1, areas[0].Tools)
	assert.Equal(t, "Shipments", areas[1].Name)
	assert.Equal(t, 2, areas[1].Tools)
	assert.Equal(t, []string{"shipment"}, areas[1].Covers)
}

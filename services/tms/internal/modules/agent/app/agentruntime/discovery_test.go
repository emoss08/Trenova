package agentruntime

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wideRun is wideRuntime wired to a scripted completion, for tests that drive
// a whole turn rather than one tool set.
func wideRun(
	t *testing.T,
	turns ...*llmports.ChatCompletionResult,
) (*Service, *scriptedCompletion, []string) {
	t.Helper()

	service, names := wideRuntime(t)
	completion := &scriptedCompletion{Turns: turns}
	rewire(service, func(p *Params) { p.Completion = completion })

	return service, completion, names
}

func requestTools(req *llmports.ChatCompletionRequest) []string {
	return specNames(req.Tools)
}

func loadedIn(req *llmports.ChatCompletionRequest) []string {
	loaded := make([]string, 0, len(req.Tools))
	for _, spec := range req.Tools {
		if !spec.Deferred {
			loaded = append(loaded, spec.Name)
		}
	}
	for _, message := range req.Messages {
		loaded = append(loaded, message.ToolReferences...)
	}

	return loaded
}

/*
find_tools works end to end: what it loads is loaded on the very next request.

Nothing tested this through Run. The tool set grew, but a test that only looked
at the set could not see whether the request the model answers next carried it.
The tool was declared deferred from the first call, and the find result names
it, so the request's tools stay as they were.
*/
func TestRun_FindToolsLoadsAToolIntoTheNextRequest(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(
		t,
		runtimetest.ToolTurn(
			runtimetools.FindToolsName,
			map[string]any{"need": "trailers due inspection"},
		),
		runtimetest.ToolTurn("equipment_list_trailers", map[string]any{}),
		runtimetest.TextTurn("Two trailers are due."),
	)

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	require.Len(t, completion.Requests, 3)
	assert.NotContains(t, loadedIn(completion.Requests[0]), "equipment_list_trailers",
		"the tool was not preselected for a greeting")
	assert.Contains(t, requestTools(completion.Requests[0]), "equipment_list_trailers",
		"it is declared, deferred, from the first call")
	assert.Contains(t, loadedIn(completion.Requests[1]), "equipment_list_trailers",
		"what find_tools loaded is loaded on the next request")
	found := completion.Requests[1].Messages[len(completion.Requests[1].Messages)-1]
	assert.Equal(t, runtimetools.FindToolsName, found.ToolName)
	assert.Contains(t, found.ToolReferences, "equipment_list_trailers", "the find result names it")
	assert.Equal(t, completion.Requests[0].Tools, completion.Requests[1].Tools)
	assert.Equal(t, "Two trailers are due.", result.Reply)
	assert.Zero(t, result.Messages[2].ToolFailed)
}

/*
What a turn loaded is still loaded on the next one.

The tool set used to be rebuilt from the new message alone. After find_tools
loaded worker_list_credentials, "yes, do that" reopened on the eight tools
that ranked first against three words that name nothing, and the tool the
conversation was about was gone.
*/
func TestNewToolSet_CarriesWhatTheLastTurnsUsedAndFound(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "say hello"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{
				ID:        "c1",
				Name:      runtimetools.FindToolsName,
				Arguments: map[string]any{"need": "driver medical card expiry"},
			},
		}},
		{Role: conversation.RoleTool, Content: "These tools are now callable"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c2", Name: "report_run", Arguments: map[string]any{}},
		}},
		{Role: conversation.RoleTool, Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "Shall I list them?"},
	}

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "yes, do that",
		History:    history,
	})

	require.True(t, set.Disclosed)
	sent := specNames(set.Specs)
	assert.Contains(t, sent, "worker_list_credentials", "what find_tools found last turn")
	assert.Contains(t, sent, "report_run", "what the model called last turn")
}

// A follow-up that names nothing is ranked with the message it answers.
func TestNewToolSet_RanksAFollowUpWithTheMessageItAnswers(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "which trucks are out of service"},
		{Role: conversation.RoleAssistant, Content: "Do you want the whole fleet or one terminal?"},
	}

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "the whole fleet",
		History:    history,
	})

	assert.Contains(t, specNames(set.Specs), "equipment_list_tractors")
}

// A run nobody is watching has nobody to ask. The spec is withheld, but a model
// that names the tool anyway used to have its question shown to no one and
// the run recorded as answered.
func TestRun_RefusesAQuestionInARunNobodyIsWatching(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(
		t,
		runtimetest.ToolTurn(
			askuser.Name,
			map[string]any{"question": "Which terminal?", "allowOther": true},
		),
		runtimetest.TextTurn("Raised it for a person."),
	)

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(names[:3]...),
		Actor:      runtimetest.Actor(),
		Input:      "check the fleet",
		Unattended: true,
	})
	require.NoError(t, err)

	assert.NotContains(t, requestTools(completion.Requests[0]), askuser.Name)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "agent_raise_exception")
}

// find_tools is not charged against the budget, so a model with a budget of
// one can still look for the tool and then use it; past the cap it is.
func TestRun_FindToolsIsNotChargedUntilTheCap(t *testing.T) {
	t.Parallel()

	turns := make([]*llmports.ChatCompletionResult, 0, toolaccess.MaxFindCalls+2)
	for range toolaccess.MaxFindCalls + 1 {
		turns = append(
			turns,
			runtimetest.ToolTurn(runtimetools.FindToolsName, map[string]any{"need": "tractors"}),
		)
	}
	turns = append(turns, runtimetest.TextTurn("done"))
	service, _, names := wideRun(t, turns...)
	definition := runtimetest.Definition(names...)
	definition.MaxToolCalls = 1

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	var searches, refused int
	for _, message := range result.Messages {
		if message.Role != conversation.RoleTool {
			continue
		}
		searches++
		if message.ToolFailed {
			refused++
			assert.Contains(t, message.Content, "searched for tools")
		}
	}
	assert.Equal(
		t,
		toolaccess.MaxFindCalls+1,
		searches,
		"the cap is reached before the budget of one",
	)
	assert.Equal(t, 1, refused)
	assert.Equal(t, 1, result.ToolCallsUsed)
}

func TestResolveFind_WithoutACatalogSaysSoRatherThanPanicking(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	rewire(service, func(p *Params) { p.Catalog = nil })
	set := &toolaccess.ToolSet{Loaded: map[string]struct{}{}, Disclosed: true}

	assert.Contains(t, findContent(t, service, set, map[string]any{"need": "tractors"}),
		"No other tools")
}

// dependentTool is an action whose arguments come from other tools.
type dependentTool struct {
	*agenttest.StubActionTool
	needs []string
}

func (t *dependentTool) Prerequisites() []string { return t.needs }

/*
A tool that needs an id brings the tool that hands the id out.

The Homepage Widget Builder held report_create_dashboard and not report_list, so it
wrote "on_time_percentage" where a report id belonged. A read that a held tool
depends on is now held with it; a write never is, since holding one write must
not grant another.
*/
func TestRun_HoldsTheReadsAHeldToolTakesItsArgumentsFrom(t *testing.T) {
	t.Parallel()

	reports := queryTool("report_list", []map[string]any{{"id": "rpt_1"}}, nil)
	query := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{reports}}
	action := &stubActionRegistry{Tools: []agenttoolports.AgentTool{
		&dependentTool{
			StubActionTool: actionTool("report_create_dashboard", agent.TierPropose, nil),
			needs:          []string{"report_list", "report_delete"},
		},
		actionTool("report_delete", agent.TierPropose, nil),
	}}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_list", map[string]any{}),
		runtimetest.ToolTurn("report_delete", map[string]any{}),
		runtimetest.TextTurn("Found one report."),
	}}
	service := newRuntime(completion, query, action, nil)
	rewire(service, func(p *Params) {
		p.Catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		})
	})

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_create_dashboard"),
		Actor:      runtimetest.Actor(),
		Input:      "build me a dashboard",
	})
	require.NoError(t, err)

	assert.Contains(t, requestTools(completion.Requests[0]), "report_list",
		"the read the dashboard tool depends on is offered with it")
	assert.Equal(t, 1, reports.Calls)
	assert.True(t, result.Messages[4].ToolFailed, "a write is never granted as a dependency")
	assert.Contains(t, result.Messages[4].Content, "not enabled for this agent")
}

type companionTool struct {
	*agenttest.StubActionTool
	brings []string
}

func (t *companionTool) Companions() []string { return t.brings }

/*
A write brings the writes its job cannot be finished without.

The Shipment Assistant held shipment_create and nothing that adds a customer,
so a rate confirmation from a broker Trenova did not know ended with "no match
was found; no shipment was created". shipment_create names customer_create,
location_create and masterdata_create_commodity as its companions: an agent holding it
holds them, each still proposed, and one switched off for the agent stays off.
*/
func TestRun_HoldsTheWritesAHeldToolNamesAsCompanions(t *testing.T) {
	t.Parallel()

	categories := queryTool("location_list_categories", []map[string]any{}, nil)
	query := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{categories}}
	action := &stubActionRegistry{Tools: []agenttoolports.AgentTool{
		&companionTool{
			StubActionTool: actionTool("shipment_create", agent.TierPropose, nil),
			brings:         []string{"customer_create", "location_create"},
		},
		actionTool("customer_create", agent.TierPropose, nil),
		&dependentTool{
			StubActionTool: actionTool("location_create", agent.TierPropose, nil),
			needs:          []string{"location_list_categories"},
		},
		actionTool("delete_customer", agent.TierPropose, nil),
	}}
	service := newRuntime(&scriptedCompletion{}, query, action, nil)
	rewire(service, func(p *Params) {
		p.Catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		})
	})

	held := service.access.HeldTools(runtimetest.Definition("shipment_create"))
	assert.Contains(t, held, "customer_create")
	assert.Contains(t, held, "location_create")
	assert.Contains(t, held, "location_list_categories",
		"a companion's own reads come with it")
	assert.NotContains(t, held, "delete_customer")

	definition := runtimetest.Definition("shipment_create")
	definition.DisabledToolNames = []string{"customer_create"}
	held = service.access.HeldTools(definition)
	assert.NotContains(t, held, "customer_create", "a companion switched off stays off")
	assert.Contains(t, held, "location_create")

	set := &toolaccess.ToolSet{
		Loaded: map[string]struct{}{},
		Allowed: service.access.HeldTools(runtimetest.Definition(
			"shipment_create",
		)),
	}
	require.True(t, service.access.Load(set, "shipment_create"))
	assert.True(t, set.Offers("customer_create"), "a companion is loaded with its tool")
	assert.True(t, set.Offers("location_create"))
	assert.True(t, set.Offers("location_list_categories"))
}

/*
A disclosed turn that calls a tool the agent does not hold is handed the
nearest ones it does, loaded, instead of a bare refusal.
*/
func TestRun_RefusalLoadsTheNearestToolsTheAgentHolds(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t,
		runtimetest.ToolTurn("search_trailers", map[string]any{"query": "53"}),
		runtimetest.TextTurn("done"),
	)
	held := make([]string, 0, len(names))
	for _, name := range names {
		if name != "worker_list" {
			held = append(held, name)
		}
	}

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(held...),
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	refusal := result.Messages[2].Content
	assert.Contains(t, refusal, "There is no tool named")
	assert.Contains(t, refusal, "equipment_list_trailers")
	assert.Contains(t, loadedIn(completion.Requests[1]), "equipment_list_trailers")
}

func TestRun_ADisclosedTurnNeverChangesItsToolArray(t *testing.T) {
	t.Parallel()

	const input = "which drivers have a medical card expiring soon"
	probe, names := wideRuntime(t)
	definition := runtimetest.Definition(names...)
	opening := specNames(probe.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      input,
	}).Specs)
	unsent := slices.DeleteFunc(slices.Clone(names), func(name string) bool {
		return slices.Contains(opening, name)
	})
	require.GreaterOrEqual(t, len(unsent), 2, "the wide agent is disclosed")
	found, called := unsent[0], unsent[1]

	service, completion, _ := wideRun(t,
		runtimetest.ToolTurn(runtimetools.FindToolsName, map[string]any{"need": found}),
		runtimetest.ToolTurn(called, map[string]any{}),
		runtimetest.ToolTurn("list_trailer_inspection_dates", map[string]any{}),
		runtimetest.TextTurn("Done."),
	)
	_, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      input,
	})
	require.NoError(t, err)
	require.Len(t, completion.Requests, 4)

	for idx := 1; idx < len(completion.Requests); idx++ {
		assert.Equal(t, completion.Requests[0].Tools, completion.Requests[idx].Tools,
			"model call %d changed the declared tools", idx+1)
	}
	declared := requestTools(completion.Requests[0])
	assert.True(t, slices.IsSorted(declared), "the declaration is sorted by name")
	for _, name := range names {
		assert.Contains(t, declared, name, "every held tool is declared from the first call")
	}
	assert.NotContains(t, loadedIn(completion.Requests[0]), found)
	assert.Contains(t, loadedIn(completion.Requests[1]), found,
		"resolveFind loads what find_tools answered")
	assert.NotContains(t, loadedIn(completion.Requests[1]), called)
	assert.Contains(t, loadedIn(completion.Requests[2]), called,
		"the runner loads a held tool the model called without it being loaded")
	assert.GreaterOrEqual(t, len(loadedIn(completion.Requests[3])),
		len(loadedIn(completion.Requests[2])),
		"unheldRefusal loads the nearest held tools for a name that does not exist")
}

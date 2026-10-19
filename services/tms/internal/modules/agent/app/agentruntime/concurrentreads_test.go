package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// batchRecorder runs every effect in process except the tool calls, which it
// answers and records as the loop handed them over: one at a time, or as reads
// it may run together.
type batchRecorder struct {
	*localEffects

	steps   []string
	earlier map[string][]string
	events  []string
}

func newBatchRecorder(t *testing.T, rt *Service) *batchRecorder {
	t.Helper()

	fx := &batchRecorder{earlier: map[string][]string{}}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: fx.record}

	return fx
}

func (fx *batchRecorder) record(event kernelports.StreamEvent) {
	switch data := event.Data.(type) {
	case agentports.AssistantToolStartedEvent:
		fx.events = append(fx.events, "started "+data.CallID)
	case agentports.AssistantToolFinishedEvent:
		fx.events = append(fx.events, "finished "+data.CallID)
	}
}

func (fx *batchRecorder) answer(call DispatchCall) ToolOutcome {
	fx.earlier[call.Call.ID] = call.Earlier

	return ToolOutcome{Content: `{"results":[]}`}
}

func (fx *batchRecorder) Dispatch(_ *Turn, call DispatchCall) ToolOutcome {
	fx.steps = append(fx.steps, "alone "+call.Call.ID)

	return fx.answer(call)
}

func (fx *batchRecorder) DispatchReads(_ *Turn, calls []DispatchCall) []ToolOutcome {
	ids := make([]string, 0, len(calls))
	outcomes := make([]ToolOutcome, 0, len(calls))
	for _, call := range calls {
		ids = append(ids, call.Call.ID)
		outcomes = append(outcomes, fx.answer(call))
	}
	fx.steps = append(fx.steps, "together "+strings.Join(ids, " "))

	return outcomes
}

func readCall(id, name string, args map[string]any) agentports.ToolCall {
	return agentports.ToolCall{ID: id, Name: name, Arguments: args}
}

type readScenario struct {
	calls  []agentports.ToolCall
	budget int
}

// driveReads runs one reply's calls through the loop against a recorder, with
// worker_list, search_customer and shipment_get held as reads and
// record_case held as a write.
func driveReads(t *testing.T, scenario readScenario) (*batchRecorder, *agentports.RunResult) {
	t.Helper()

	rt := newRuntime(
		&scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			callsTurn(scenario.calls...),
			textTurn("Done."),
		}},
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{
			queryTool("worker_list", map[string]any{"results": []any{}}, nil),
			queryTool("search_customer", map[string]any{"results": []any{}}, nil),
			queryTool("shipment_get", map[string]any{"id": "shp_1"}, nil),
		}},
		&stubActionRegistry{Tools: []agentports.AgentTool{
			actionTool("record_case", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	definition := testDefinition("worker_list", "search_customer", "shipment_get", "record_case")
	if scenario.budget > 0 {
		definition.MaxToolCalls = scenario.budget
	}

	fx := newBatchRecorder(t, rt)
	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Look it all up.",
	}), fx)
	require.NoError(t, err)

	return fx, result
}

func resultCallIDs(result *agentports.RunResult) []string {
	ids := make([]string, 0, len(result.Messages))
	for idx := range result.Messages {
		if result.Messages[idx].ToolCallID != "" {
			ids = append(ids, result.Messages[idx].ToolCallID)
		}
	}

	return ids
}

func resultContent(result *agentports.RunResult, callID string) string {
	for idx := range result.Messages {
		if result.Messages[idx].ToolCallID == callID {
			return result.Messages[idx].Content
		}
	}

	return ""
}

/*
A model that asks for three lookups in one reply waits for the slowest of
them, not for all three end to end. What the model, the stream and the
transcript read still comes back in the order it asked.
*/
func TestDrive_RunsAReplysReadsTogether(t *testing.T) {
	t.Parallel()

	fx, result := driveReads(t, readScenario{calls: []agentports.ToolCall{
		readCall("c1", "worker_list", map[string]any{"status": "Active"}),
		readCall("c2", "search_customer", map[string]any{"query": "Acme"}),
		readCall("c3", "shipment_get", map[string]any{"id": "shp_1"}),
	}})

	assert.Equal(t, []string{"together c1 c2 c3"}, fx.steps)
	assert.Equal(t, []string{"c1", "c2", "c3"}, resultCallIDs(result))
	assert.Equal(t, []string{
		"started c1", "started c2", "started c3",
		"finished c1", "finished c2", "finished c3",
	}, fx.events)
	assert.Equal(t, 3, result.ToolCallsUsed)
}

// A write runs on its own, after every read before it has settled and before
// any read after it starts, since it may change what they return.
func TestDrive_AWriteRunsBetweenTheReadsAroundIt(t *testing.T) {
	t.Parallel()

	fx, result := driveReads(t, readScenario{calls: []agentports.ToolCall{
		readCall("c1", "worker_list", map[string]any{"status": "Active"}),
		readCall("c2", "search_customer", map[string]any{"query": "Acme"}),
		readCall("c3", "record_case", map[string]any{"subject": "late"}),
		readCall("c4", "worker_list", map[string]any{"status": "Inactive"}),
		readCall("c5", "search_customer", map[string]any{"query": "Globex"}),
	}})

	assert.Equal(t, []string{"together c1 c2", "alone c3", "together c4 c5"}, fx.steps)
	assert.Equal(t, []string{"c1", "c2", "c3", "c4", "c5"}, resultCallIDs(result))
}

/*
A get_ call's card is bunched with the turn's earlier cards of the same tool,
read back by call id. Two fetches of one kind never run together, and the
second is told about the first, so the cards still join one table.
*/
func TestDrive_FetchesOfOneKindRunApartAndStillBunch(t *testing.T) {
	t.Parallel()

	fx, _ := driveReads(t, readScenario{calls: []agentports.ToolCall{
		readCall("c1", "shipment_get", map[string]any{"id": "shp_1"}),
		readCall("c2", "shipment_get", map[string]any{"id": "shp_2"}),
		readCall("c3", "worker_list", map[string]any{"status": "Active"}),
	}})

	assert.Equal(t, []string{"alone c1", "together c2 c3"}, fx.steps)
	assert.Equal(t, []string{"c1"}, fx.earlier["c2"], "the second fetch knows the first")
}

func TestDrive_AnswersARepeatedReadInOneReplyFromTheFirst(t *testing.T) {
	t.Parallel()

	args := map[string]any{"status": "Active"}
	fx, result := driveReads(t, readScenario{calls: []agentports.ToolCall{
		readCall("c1", "worker_list", args),
		readCall("c2", "worker_list", args),
	}})

	assert.Equal(t, []string{"alone c1"}, fx.steps)
	require.Equal(t, []string{"c1", "c2"}, resultCallIDs(result))
	assert.Equal(t, repeatedRead("worker_list"), resultContent(result, "c2"))
}

func TestDrive_HoldsAReadBatchToTheToolBudget(t *testing.T) {
	t.Parallel()

	fx, result := driveReads(t, readScenario{budget: 2, calls: []agentports.ToolCall{
		readCall("c1", "worker_list", map[string]any{"status": "Active"}),
		readCall("c2", "search_customer", map[string]any{"query": "Acme"}),
		readCall("c3", "shipment_get", map[string]any{"id": "shp_1"}),
	}})

	assert.Equal(t, []string{"together c1 c2"}, fx.steps)
	assert.Equal(t, 2, result.ToolCallsUsed)
	assert.Contains(t, resultContent(result, "c3"), "tool budget of 2 is spent")
}

func TestDrive_RunsAtMostEightReadsAtOnce(t *testing.T) {
	t.Parallel()

	calls := make([]agentports.ToolCall, 0, 10)
	ids := make([]string, 0, 10)
	for idx := range 10 {
		id := "c" + string(rune('a'+idx))
		ids = append(ids, id)
		calls = append(calls, readCall(id, "worker_list", map[string]any{"page": idx}))
	}

	fx, _ := driveReads(t, readScenario{budget: 20, calls: calls})

	assert.Equal(t, []string{
		"together " + strings.Join(ids[:8], " "),
		"together " + strings.Join(ids[8:], " "),
	}, fx.steps)
}

// The reads are taken when the turn opens and cross the activity boundary
// with it.
func TestRestoreTurn_BatchesOnlyTheReadsTheTurnOpenedWith(t *testing.T) {
	t.Parallel()

	rt := newRuntime(
		nil,
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{
			queryTool("worker_list", map[string]any{"results": []any{}}, nil),
			queryTool("search_customer", map[string]any{"results": []any{}}, nil),
		}},
		&stubActionRegistry{Tools: []agentports.AgentTool{
			actionTool("record_case", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	request := &agentports.RunRequest{
		Definition: testDefinition("worker_list", "search_customer", "record_case"),
		Actor:      testActor(),
		Input:      "Who is free?",
	}
	state := rt.OpenTurn(t.Context(), request).State()
	assert.Equal(t, []string{"worker_list", "search_customer"}, state.Reads,
		"a write is never a read, whatever it is held with")

	encoded, err := sonic.Marshal(state)
	require.NoError(t, err)
	var carried TurnState
	require.NoError(t, sonic.Unmarshal(encoded, &carried))
	require.Equal(t, state.Reads, carried.Reads)

	drive := func(state TurnState) []string {
		rt.completion = &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			callsTurn(
				readCall("c1", "worker_list", map[string]any{"status": "Active"}),
				readCall("c2", "search_customer", map[string]any{"query": "Acme"}),
			),
			textTurn("Done."),
		}}
		fx := newBatchRecorder(t, rt)
		_, err := rt.Drive(rt.RestoreTurn(request, state), fx)
		require.NoError(t, err)

		return fx.steps
	}

	assert.Equal(t, []string{"together c1 c2"}, drive(carried))
}

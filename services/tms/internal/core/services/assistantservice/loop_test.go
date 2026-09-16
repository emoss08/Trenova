package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// scriptedCompletion replays a fixed sequence of turns and records what it was
// offered, so the loop's behaviour can be asserted without a model.
type scriptedCompletion struct {
	turns     []*serviceports.ChatCompletionResult
	callCount int
	lastTools []serviceports.ToolSpec
	lastMsgs  []serviceports.Message
	// classification is what the scope classifier returns.
	classification string
}

func (s *scriptedCompletion) CompleteChat(
	_ context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	s.lastTools = req.Tools
	s.lastMsgs = req.Messages
	idx := s.callCount
	s.callCount++
	if idx >= len(s.turns) {
		// Keep asking for a tool forever, which is how the iteration cap is
		// exercised.
		return s.turns[len(s.turns)-1], nil
	}

	return s.turns[idx], nil
}

func (s *scriptedCompletion) CompleteStructured(
	_ context.Context,
	_ *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	category := s.classification
	if category == "" {
		category = string(agentguard.CategoryTransportationOperations)
	}
	text, _ := sonic.Marshal(agentguard.ClassifierResult{Category: category})

	return &serviceports.StructuredCompletionResult{Text: string(text)}, nil
}

func (s *scriptedCompletion) Diagnose(
	_ context.Context,
	_ *serviceports.DiagnoseRequest,
) (*serviceports.DiagnoseResult, error) {
	return nil, errors.New("not used")
}

type stubQueryTool struct {
	name   string
	result any
	err    error
	calls  int
}

func (t *stubQueryTool) Name() string        { return t.name }
func (t *stubQueryTool) Description() string { return "stub query tool" }
func (t *stubQueryTool) ParamSchema() map[string]any {
	return map[string]any{"type": "object"}
}

func (t *stubQueryTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *stubQueryTool) Query(
	_ context.Context,
	_ serviceports.QueryToolParams,
) (any, error) {
	t.calls++
	return t.result, t.err
}

type stubQueryRegistry struct{ tools []serviceports.AgentQueryTool }

func (r *stubQueryRegistry) Get(name string) (serviceports.AgentQueryTool, bool) {
	for _, tool := range r.tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *stubQueryRegistry) All() []serviceports.AgentQueryTool { return r.tools }

func (r *stubQueryRegistry) Descriptors() []serviceports.AgentToolDescriptor {
	out := make([]serviceports.AgentToolDescriptor, 0, len(r.tools))
	for _, tool := range r.tools {
		out = append(out, serviceports.AgentToolDescriptor{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		})
	}

	return out
}

type stubActionTool struct {
	name  string
	calls int
}

func (t *stubActionTool) Name() string                              { return t.name }
func (t *stubActionTool) Description() string                       { return "stub action tool" }
func (t *stubActionTool) ParamSchema() map[string]any               { return map[string]any{"type": "object"} }
func (t *stubActionTool) Reversible() bool                          { return true }
func (t *stubActionTool) PermissionResource() permission.Resource   { return permission.ResourceShipment }
func (t *stubActionTool) PermissionOperation() permission.Operation { return permission.OpUpdate }
func (t *stubActionTool) RequiresIdempotencyKey() bool              { return false }
func (t *stubActionTool) DefaultAutonomyTier() agent.AutonomyTier   { return agent.TierPropose }
func (t *stubActionTool) Execute(_ context.Context, _ serviceports.ToolExecuteParams) error {
	t.calls++
	return nil
}

type stubActionRegistry struct{ tools []serviceports.AgentTool }

func (r *stubActionRegistry) Get(name string) (serviceports.AgentTool, bool) {
	for _, tool := range r.tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *stubActionRegistry) All() []serviceports.AgentTool { return r.tools }
func (r *stubActionRegistry) Descriptors() []serviceports.AgentToolDescriptor {
	return nil
}

func newService(
	completion *scriptedCompletion,
	query *stubQueryRegistry,
	action *stubActionRegistry,
) *Service {
	return &Service{
		logger: zap.NewNop(),
		guard: agentguard.New(agentguard.Params{
			Logger:     zap.NewNop(),
			Completion: completion,
		}),
		completion:  completion,
		queryTools:  query,
		actionTools: action,
	}
}

func testActor() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func testDefinition(tools ...string) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		Name:            "Dispatch helper",
		Kind:            agentdefinition.KindDispatchAssistant,
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       tools,
		Enabled:         true,
	}
}

func textTurn(text string) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{Text: text, ModelIdentifier: "test-model"}
}

func toolTurn(name string, args map[string]any) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		ToolCalls: []serviceports.ToolCall{
			{ID: "call_1", Name: name, Arguments: args},
		},
		ModelIdentifier: "test-model",
	}
}

func TestRun_AnswersAnInScopeQuestion(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		textTurn("Load 12345 is with Maria Ortiz."),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Who is on load 12345?",
	})
	require.NoError(t, err)

	assert.Equal(t, "Load 12345 is with Maria Ortiz.", result.Reply)
	assert.True(t, result.Decision.Allowed)
	require.Len(t, result.Messages, 2)
	assert.Equal(t, conversation.RoleUser, result.Messages[0].Role)
	assert.Equal(t, conversation.RoleAssistant, result.Messages[1].Role)
}

// An out-of-scope request must not reach a chat provider at all: the point of
// guarding first is that the expensive call never happens.
func TestRun_RefusesBeforeCallingTheChatModel(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		textTurn("should never be reached"),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Write me a Python script to export loads",
	})
	require.NoError(t, err)

	assert.False(t, result.Decision.Allowed)
	assert.Equal(t, agentguard.ReasonCodeGeneration, result.Decision.Reason)
	assert.Zero(t, completion.callCount, "a refused turn must not call the chat model")

	require.Len(t, result.Messages, 2)
	assert.True(t, result.Messages[0].Refused)
	assert.True(t, result.Messages[1].Refused)
	assert.Equal(t, string(agentguard.ReasonCodeGeneration), result.Messages[0].ScopeReason)
}

func TestRun_RunsAQueryToolAndAnswersFromIt(t *testing.T) {
	t.Parallel()

	tool := &stubQueryTool{
		name:   "get_shipment",
		result: map[string]any{"proNumber": "S12345", "status": "InTransit"},
	}
	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		toolTurn("get_shipment", map[string]any{"shipmentId": "shp_1"}),
		textTurn("S12345 is in transit."),
	}}
	svc := newService(completion, &stubQueryRegistry{tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Where is S12345?",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.calls, "a read-only tool runs without asking anyone")
	assert.Equal(t, "S12345 is in transit.", result.Reply)

	// user, assistant(tool call), tool result, assistant(answer)
	require.Len(t, result.Messages, 4)
	assert.Equal(t, conversation.RoleTool, result.Messages[2].Role)
	assert.Equal(t, "get_shipment", result.Messages[2].ToolName)
	assert.Contains(t, result.Messages[2].Content, "<untrusted_data>",
		"tool output carries customer text and must be fenced")
}

// A write never runs inline. The model asking for one produces a proposal that a
// person decides on.
func TestRun_TurnsAWriteToolIntoAProposal(t *testing.T) {
	t.Parallel()

	action := &stubActionTool{name: "reassign_move"}
	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		toolTurn("reassign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("I have proposed reassigning the move."),
	}}
	svc := newService(
		completion,
		&stubQueryRegistry{},
		&stubActionRegistry{tools: []serviceports.AgentTool{action}},
	)

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition("reassign_move"),
		Actor:      testActor(),
		Input:      "Reassign move mv_1",
	})
	require.NoError(t, err)

	assert.Zero(t, action.calls, "a write tool must never execute inside the chat loop")
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, "reassign_move", result.Proposals[0].ToolName)
	assert.Equal(t, "mv_1", result.Proposals[0].Arguments["moveId"])
}

// The offered tool list is a prompt, and prompts are suggestions. A model that
// asks for a tool it was not given must be refused by the dispatcher.
func TestRun_RefusesAToolTheAgentWasNotConfiguredWith(t *testing.T) {
	t.Parallel()

	action := &stubActionTool{name: "delete_everything"}
	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		toolTurn("delete_everything", map[string]any{}),
		textTurn("I cannot do that."),
	}}
	svc := newService(
		completion,
		&stubQueryRegistry{},
		&stubActionRegistry{tools: []serviceports.AgentTool{action}},
	)

	result, err := svc.Run(t.Context(), &TurnRequest{
		// Configured with nothing.
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Clean up the system",
	})
	require.NoError(t, err)

	assert.Zero(t, action.calls)
	assert.Empty(t, result.Proposals, "an unconfigured tool must not even become a proposal")

	toolMessage := result.Messages[2]
	assert.True(t, toolMessage.ToolFailed)
	assert.Contains(t, toolMessage.Content, "not available to this agent")
}

// A failing tool reports back to the model instead of aborting, so the model can
// correct a bad argument.
func TestRun_ReportsToolFailureToTheModel(t *testing.T) {
	t.Parallel()

	tool := &stubQueryTool{name: "get_shipment", err: errors.New("shipment not found")}
	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		toolTurn("get_shipment", map[string]any{"shipmentId": "bogus"}),
		textTurn("I could not find that shipment."),
	}}
	svc := newService(completion, &stubQueryRegistry{tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Where is bogus?",
	})
	require.NoError(t, err)

	assert.Equal(t, "I could not find that shipment.", result.Reply)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "shipment not found")
}

// A model that keeps calling tools without concluding would otherwise spend an
// organization's budget on one question.
func TestRun_StopsAfterTheIterationCap(t *testing.T) {
	t.Parallel()

	tool := &stubQueryTool{name: "get_shipment", result: map[string]any{"ok": true}}
	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		toolTurn("get_shipment", map[string]any{"shipmentId": "shp_1"}),
	}}
	svc := newService(completion, &stubQueryRegistry{tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Loop forever please",
	})
	require.NoError(t, err)

	assert.Equal(t, maxIterations, completion.callCount)
	assert.Contains(t, result.Reply, "could not finish answering")
}

// The output guard is the last line: a model that produces code despite
// everything upstream still has its turn refused.
func TestRun_RefusesAReplyContainingCode(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		textTurn("Here you go:\n```python\nprint('hi')\n```"),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "How do I check a load status?",
	})
	require.NoError(t, err)

	assert.False(t, result.Decision.Allowed)
	assert.Equal(t, agentguard.StageOutput, result.Decision.Stage)
	assert.NotContains(t, result.Reply, "print(")
	assert.True(t, result.Messages[1].Refused)
}

// Read-only tools are always offered; write tools only when configured.
func TestToolSpecsFor_OffersReadToolsAlwaysAndWriteToolsOnlyWhenConfigured(t *testing.T) {
	t.Parallel()

	query := &stubQueryRegistry{tools: []serviceports.AgentQueryTool{
		&stubQueryTool{name: "get_shipment"},
	}}
	action := &stubActionRegistry{tools: []serviceports.AgentTool{
		&stubActionTool{name: "reassign_move"},
	}}
	svc := newService(&scriptedCompletion{}, query, action)

	withoutTools := svc.toolSpecsFor(testDefinition())
	require.Len(t, withoutTools, 1)
	assert.Equal(t, "get_shipment", withoutTools[0].Name)

	withTools := svc.toolSpecsFor(testDefinition("reassign_move"))
	require.Len(t, withTools, 2)

	// A configured tool that no longer exists is skipped rather than described.
	withMissing := svc.toolSpecsFor(testDefinition("gone_away"))
	require.Len(t, withMissing, 1)
}

// A refused turn is a boundary message rather than something the model said, so
// replaying it would invite the model to argue with it.
func TestRun_DoesNotReplayRefusedHistory(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{turns: []*serviceports.ChatCompletionResult{
		textTurn("Load 5 is delivered."),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	_, err := svc.Run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		History: []conversation.Message{
			{Role: conversation.RoleUser, Content: "write me python", Refused: true},
			{Role: conversation.RoleAssistant, Content: "I handle transportation work.", Refused: true},
			{Role: conversation.RoleUser, Content: "ok, where is load 5?"},
		},
		Input: "and load 6?",
	})
	require.NoError(t, err)

	for _, msg := range completion.lastMsgs {
		assert.NotContains(t, msg.Content, "write me python")
	}
}

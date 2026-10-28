package agentflow

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/contrib/workflowstreams"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const testTurnWorkflow = "agentflow-test-turn"

// turnResult carries the outcome and the error together. A workflow that
// returns an error drops its result, and the point of several of these tests
// is what the turn had done by the time it failed.
type turnResult struct {
	Outcome *Outcome `json:"outcome"`
	Err     string   `json:"err"`
}

type harness struct {
	env        *testsuite.TestWorkflowEnvironment
	runtime    *agentruntime.Service
	activities *Activities
}

type harnessParams struct {
	query  []agenttoolports.AgentQueryTool
	action []agenttoolports.AgentTool
	ledger agentruntimeports.RunStepLedger
	// budgets reads the organization's money limits for a money write.
	budgets agentports.AgentBudgetService
	// delegates opens another agent's turn when the run's agent hands it a
	// task.
	delegates DelegateOpener
	// wrapTool, when set, stands between the worker and the real tool
	// activity, so a test can fail an attempt after the tool has run.
	wrapTool func(
		run func(context.Context, converter.EncodedValues) (*ToolResult, error),
	) func(context.Context, converter.EncodedValues) (*ToolResult, error)
}

func newHarness(t *testing.T, p harnessParams) *harness {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	rt := agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agenttest.ScriptedCompletion{},
		QueryTools:  &agenttest.StubQueryRegistry{Tools: p.query},
		ActionTools: &agenttest.StubActionRegistry{Tools: p.action},
		Permissions: &agenttest.StubPermissions{},
		Budgets:     p.budgets,
	})
	acts := NewActivities(ActivitiesParams{
		Logger:    zap.NewNop(),
		Runtime:   rt,
		Steps:     p.ledger,
		Delegates: p.delegates,
	})

	env.RegisterActivity(acts)
	tool := acts.runTool
	if p.wrapTool != nil {
		tool = p.wrapTool(tool)
	}
	env.RegisterDynamicActivity(tool, activity.DynamicRegisterOptions{})
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, rc RunContext, state agentruntime.TurnState) (*turnResult, error) {
			stream, err := workflowstreams.NewWorkflowStream(ctx, nil)
			if err != nil {
				return nil, err
			}

			outcome, err := Run(ctx, rt, stream.Topic(EventsTopic), rc, state)
			result := &turnResult{Outcome: outcome}
			if err != nil {
				result.Err = err.Error()
			}

			return result, nil
		},
		workflow.RegisterOptions{Name: testTurnWorkflow},
	)

	return &harness{env: env, runtime: rt, activities: acts}
}

// replies scripts the model: each call to the model activity answers with the
// next reply in turn.
func (h *harness) replies(replies ...*llmports.ChatCompletionResult) {
	for _, reply := range replies {
		h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
			Return(&agentruntime.ModelReply{Completion: reply}, nil).Once()
	}
}

func (h *harness) run(t *testing.T, rc RunContext) *turnResult {
	t.Helper()

	state := h.runtime.OpenTurn(t.Context(), rc.request()).State()
	h.env.ExecuteWorkflow(testTurnWorkflow, rc, state)

	require.True(t, h.env.IsWorkflowCompleted())
	require.NoError(t, h.env.GetWorkflowError())

	var result turnResult
	require.NoError(t, h.env.GetWorkflowResult(&result))
	h.env.AssertExpectations(t)

	return &result
}

func runContext(tools ...string) RunContext {
	definition := &agentdefinition.Definition{
		Name:            "Dispatch helper",
		Instructions:    "Help dispatch.",
		AutonomyCeiling: agent.TierAutoExecute,
		ToolNames:       tools,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	return RunContext{
		Definition: definition,
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Input:       "Where is load 12345?",
		PriorityKey: agentflowcontract.PriorityInteractive,
	}
}

func textReply(text string) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{Text: text, ModelIdentifier: "test-model"}
}

func toolReply(name string, args map[string]any) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		ToolCalls:       []llmports.ToolCall{{ID: "call_1", Name: name, Arguments: args}},
		ModelIdentifier: "test-model",
	}
}

func eventNames(outcome *Outcome) []string {
	names := make([]string, 0, len(outcome.Events))
	for _, event := range outcome.Events {
		names = append(names, event.Event)
	}

	return names
}

func TestRunAnswersWithoutTools(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	h.replies(textReply("Load 12345 is in Memphis."))

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	assert.Equal(t, "Load 12345 is in Memphis.", result.Outcome.Result.Reply)
	require.Len(t, result.Outcome.Result.Messages, 2)
	assert.Equal(t, conversation.RoleAssistant, result.Outcome.Result.Messages[1].Role)
	assert.Empty(t, result.Outcome.Events,
		"a plain answer streams as deltas from the model activity, and those are not kept")
}

func TestRunCallsAToolAsItsOwnActivity(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"status": "InTransit", "city": "Memphis"},
	}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})
	h.replies(
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("It is in transit near Memphis."),
	)

	var started []string
	h.env.SetOnActivityStartedListener(
		func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
			started = append(started, info.ActivityType.Name)
		},
	)

	result := h.run(t, runContext("shipment_get"))

	require.Empty(t, result.Err)
	assert.Equal(t, 1, lookup.Calls)
	assert.Equal(t, "12345", lookup.LastParams.Params["proNumber"])
	assert.Contains(t, started, "shipment_get",
		"the tool is scheduled under its own name, so it reads as itself in the UI")
	assert.Equal(t, "It is in transit near Memphis.", result.Outcome.Result.Reply)
	assert.Equal(t, 1, result.Outcome.Result.ToolCallsUsed)
	assert.Equal(t,
		[]string{
			assistantports.AssistantEventMessage,
			assistantports.AssistantEventToolStarted,
			assistantports.AssistantEventToolFinished,
		},
		eventNames(result.Outcome),
	)

	tool := result.Outcome.Result.Messages[2]
	assert.Equal(t, conversation.RoleTool, tool.Role)
	assert.Contains(t, tool.Content, "Memphis", "the tool's answer reaches the transcript")
	assert.False(t, tool.ToolFailed)
}

func TestRunToolBindsTheRunsTenantForTheDatabase(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"status": "InTransit"},
	}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})
	h.replies(
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("It is in transit."),
	)

	rc := runContext("shipment_get")
	result := h.run(t, rc)

	require.Empty(t, result.Err)
	require.Equal(t, 1, lookup.Calls)
	assert.Equal(t, rc.Actor.TenantInfo().DBTenant(), lookup.LastTenant,
		"a dynamic activity's input is opaque to the tenant interceptor, so the tool "+
			"activity binds the run's tenant itself")
}

func TestRunToolRefusesACallWithNoTenant(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{ToolName: "shipment_get"}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})
	h.replies(
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("I could not look that up."),
	)

	rc := runContext("shipment_get")
	rc.Actor.OrganizationID = pulid.Nil
	result := h.run(t, rc)

	require.Empty(t, result.Err)
	assert.Zero(t, lookup.Calls, "a tool never runs without a tenant to scope it")
}

// A tool that cannot be run at all, even after its retries, is an answer the
// model can work with. The turn carries on and the model is told it did not
// happen.
func TestRunTellsTheModelWhenAToolCouldNotBeRun(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{ToolName: "shipment_get"}
	attempts := 0
	h := newHarness(t, harnessParams{
		query: []agenttoolports.AgentQueryTool{lookup},
		wrapTool: func(
			func(context.Context, converter.EncodedValues) (*ToolResult, error),
		) func(context.Context, converter.EncodedValues) (*ToolResult, error) {
			return func(context.Context, converter.EncodedValues) (*ToolResult, error) {
				attempts++
				return nil, errors.New("database is down")
			}
		},
	})
	h.replies(
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("I could not look that up just now."),
	)

	result := h.run(t, runContext("shipment_get"))

	require.Empty(t, result.Err)
	assert.Equal(t, 3, attempts, "the tool is retried before it is given up on")
	assert.Equal(t, "I could not look that up just now.", result.Outcome.Result.Reply)

	tool := result.Outcome.Result.Messages[2]
	assert.True(t, tool.ToolFailed)
	// A tool that never reported back may still have made its change before
	// the wait ended, so the model is told it is unconfirmed, not that it did
	// not happen.
	assert.Contains(t, tool.Content, "did not report back")
	assert.Contains(t, tool.Content, "unconfirmed")
	assert.NotContains(t, tool.Content, "did not happen")
}

// A stopped turn can end the wait on a write that already landed. Saying it
// did not happen would put a false record in the transcript.
func TestUnsettledToolOutcome_NeverSaysAStoppedWriteDidNotHappen(t *testing.T) {
	t.Parallel()

	stopped := unsettledToolOutcome("dispatch_assign_move", temporal.NewCanceledError())
	assert.True(t, stopped.Failed)
	assert.Contains(t, stopped.Content, "stopped before it reported back")
	assert.Contains(t, stopped.Content, "unconfirmed")
	assert.NotContains(t, stopped.Content, "did not happen")
}

// A model call that fails for good ends the turn, and what the turn had done
// comes back with the error, so a write it made is not lost with it.
func TestRunReturnsWhatTheTurnDidWhenTheModelFails(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{ToolName: "shipment_get", Result: "found"}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})
	h.replies(toolReply("shipment_get", map[string]any{"proNumber": "12345"}))
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("bad request", modelcall.ErrTypeModelRejected, nil)).
		Once()

	result := h.run(t, runContext("shipment_get"))

	assert.Contains(t, result.Err, "bad request")
	require.NotNil(t, result.Outcome.Result)
	assert.Equal(t, 1, result.Outcome.Result.ToolCallsUsed)
	assert.Equal(t, 1, lookup.Calls)
}

func TestRunCarriesAProposalBackAcrossTheActivity(t *testing.T) {
	t.Parallel()

	move := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierPropose}
	h := newHarness(t, harnessParams{action: []agenttoolports.AgentTool{move}})
	h.replies(
		toolReply("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		textReply("I have proposed the assignment."),
	)

	result := h.run(t, runContext("dispatch_assign_move"))

	require.Empty(t, result.Err)
	assert.Zero(t, move.Calls, "a proposal is not run inside the turn")
	require.Len(t, result.Outcome.Result.Actions, 1)
	assert.Equal(t, "mv_1", result.Outcome.Result.Actions[0].Arguments["moveId"])
	assert.False(t, result.Outcome.Result.Actions[0].Executed)
}

// memoryLedger answers a claim the way the real ledger does: fresh the first
// time, and with the recorded outcome once the step has settled.
type memoryLedger struct {
	mu      sync.Mutex
	settled map[string]agentruntimeports.RunStepOutcome
	claims  int
}

func (l *memoryLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentruntimeports.RunStep,
) (agentruntimeports.StepVerdict, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.claims++
	if outcome, ok := l.settled[step.Key]; ok {
		return agentruntimeports.StepVerdict{
			State:   agentruntimeports.StepCompleted,
			Outcome: outcome,
		}, nil
	}

	return agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh}, nil
}

func (l *memoryLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentruntimeports.RunStep,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.settled[step.Key] = step.Outcome

	return nil
}

func (*memoryLedger) Record(
	context.Context,
	pagination.TenantInfo,
	agentruntimeports.RunStep,
) error {
	return nil
}

func (*memoryLedger) Loaded(
	context.Context,
	pagination.TenantInfo,
	agentruntimeports.RunStepOwner,
) ([]agentruntimeports.RunStep, error) {
	return nil, nil
}

// A write whose activity is lost after the write landed is retried, and the
// retry is told the first attempt's answer instead of writing again.
func TestRunNeverWritesTwiceWhenAToolIsRetried(t *testing.T) {
	t.Parallel()

	move := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	ledger := &memoryLedger{settled: map[string]agentruntimeports.RunStepOutcome{}}
	h := newHarness(t, harnessParams{
		action: []agenttoolports.AgentTool{move},
		ledger: ledger,
		wrapTool: func(
			run func(context.Context, converter.EncodedValues) (*ToolResult, error),
		) func(context.Context, converter.EncodedValues) (*ToolResult, error) {
			return func(ctx context.Context, args converter.EncodedValues) (*ToolResult, error) {
				result, err := run(ctx, args)
				if activity.GetInfo(ctx).Attempt == 1 {
					return nil, errors.New("worker lost after the write")
				}

				return result, err
			}
		},
	})
	h.replies(
		toolReply("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		textReply("Assigned."),
	)

	rc := runContext("dispatch_assign_move")
	rc.RunID = pulid.MustNew("arun_")
	rc.StepOwner = agentruntimeports.RunStepOwner{
		Kind: agentruntimeports.RunStepOwnerAgentRun,
		ID:   rc.RunID,
	}

	result := h.run(t, rc)

	require.Empty(t, result.Err)
	assert.Equal(t, 1, move.Calls, "the retry must not run the write again")
	assert.Equal(t, 2, ledger.claims, "both attempts claimed the same step")
	require.Len(t, result.Outcome.Result.Actions, 1)
	assert.True(t, result.Outcome.Result.Actions[0].Executed)
}

func TestToolActivityRefusesWhatIsNotATool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		activity string
		call     string
		errType  string
	}{
		{
			name:     "an unknown name",
			activity: "SomeMisspelledActivity",
			call:     "SomeMisspelledActivity",
			errType:  ErrTypeUnknownTool,
		},
		{name: "a call to a different tool", activity: "dispatch_assign_move", call: "shipment_get",
			errType: ErrTypeBadToolInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			move := &agenttest.StubActionTool{ToolName: "dispatch_assign_move"}
			h := newHarness(t, harnessParams{action: []agenttoolports.AgentTool{move}})
			h.env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, name string, in *ToolInput) error {
					ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
						StartToCloseTimeout: toolTimeout,
						RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 5},
					})

					return workflow.ExecuteActivity(ctx, name, in).Get(ctx, nil)
				},
				workflow.RegisterOptions{Name: "agentflow-test-call"},
			)

			h.env.ExecuteWorkflow("agentflow-test-call", tt.activity, &ToolInput{
				Run: runContext("dispatch_assign_move"),
				Call: agentruntime.DispatchCall{
					Call: llmports.ToolCall{ID: "call_1", Name: tt.call},
				},
			})

			require.True(t, h.env.IsWorkflowCompleted())
			var appErr *temporal.ApplicationError
			require.ErrorAs(t, h.env.GetWorkflowError(), &appErr)
			assert.Equal(t, tt.errType, appErr.Type())
			assert.True(t, appErr.NonRetryable(), "it is refused once, not retried")
			assert.Zero(t, move.Calls)
		})
	}
}

func TestRunAsksACutOffCallAgainWithMoreRoom(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	cut := &llmports.ChatCompletionResult{
		Text:            "Transferring.",
		Truncated:       true,
		OutputLimit:     1024,
		CutOffCall:      &llmports.CutOffToolCall{Name: "billing_transfer"},
		ModelIdentifier: "z-ai/glm-5.3",
	}
	replies := []*llmports.ChatCompletionResult{cut, textReply("Nothing is waiting.")}
	var (
		mu      sync.Mutex
		budgets []int
	)
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelCallInput) (*agentruntime.ModelReply, error) {
			mu.Lock()
			defer mu.Unlock()
			budgets = append(budgets, in.Request.MaxTokens)

			return &agentruntime.ModelReply{Completion: replies[len(budgets)-1]}, nil
		}).Times(2)

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	assert.Equal(t, []int{0, 2048}, budgets)
	assert.Equal(t, "Nothing is waiting.", result.Outcome.Result.Reply)
	assert.Contains(t, eventNames(result.Outcome), assistantports.AssistantEventRetrying)
}

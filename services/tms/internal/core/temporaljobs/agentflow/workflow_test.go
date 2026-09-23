package agentflow

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
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
	query  []serviceports.AgentQueryTool
	action []serviceports.AgentTool
	ledger serviceports.RunStepLedger
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
		Completion:  &agentruntimetest.ScriptedCompletion{},
		QueryTools:  &agentruntimetest.StubQueryRegistry{Tools: p.query},
		ActionTools: &agentruntimetest.StubActionRegistry{Tools: p.action},
		Permissions: &agentruntimetest.StubPermissions{},
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
func (h *harness) replies(replies ...*serviceports.ChatCompletionResult) {
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
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Input:       "Where is load 12345?",
		PriorityKey: PriorityInteractive,
	}
}

func textReply(text string) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{Text: text, ModelIdentifier: "test-model"}
}

func toolReply(name string, args map[string]any) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		ToolCalls:       []serviceports.ToolCall{{ID: "call_1", Name: name, Arguments: args}},
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

	lookup := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"status": "InTransit", "city": "Memphis"},
	}
	h := newHarness(t, harnessParams{query: []serviceports.AgentQueryTool{lookup}})
	h.replies(
		toolReply("get_shipment", map[string]any{"proNumber": "12345"}),
		textReply("It is in transit near Memphis."),
	)

	var started []string
	h.env.SetOnActivityStartedListener(
		func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
			started = append(started, info.ActivityType.Name)
		},
	)

	result := h.run(t, runContext("get_shipment"))

	require.Empty(t, result.Err)
	assert.Equal(t, 1, lookup.Calls)
	assert.Equal(t, "12345", lookup.LastParams.Params["proNumber"])
	assert.Contains(t, started, "get_shipment",
		"the tool is scheduled under its own name, so it reads as itself in the UI")
	assert.Equal(t, "It is in transit near Memphis.", result.Outcome.Result.Reply)
	assert.Equal(t, 1, result.Outcome.Result.ToolCallsUsed)
	assert.Equal(t,
		[]string{
			serviceports.AssistantEventMessage,
			serviceports.AssistantEventToolStarted,
			serviceports.AssistantEventToolFinished,
		},
		eventNames(result.Outcome),
	)

	tool := result.Outcome.Result.Messages[2]
	assert.Equal(t, conversation.RoleTool, tool.Role)
	assert.Contains(t, tool.Content, "Memphis", "the tool's answer reaches the transcript")
	assert.False(t, tool.ToolFailed)
}

// A tool that cannot be run at all, even after its retries, is an answer the
// model can work with. The turn carries on and the model is told it did not
// happen.
func TestRunTellsTheModelWhenAToolCouldNotBeRun(t *testing.T) {
	t.Parallel()

	lookup := &agentruntimetest.StubQueryTool{ToolName: "get_shipment"}
	attempts := 0
	h := newHarness(t, harnessParams{
		query: []serviceports.AgentQueryTool{lookup},
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
		toolReply("get_shipment", map[string]any{"proNumber": "12345"}),
		textReply("I could not look that up just now."),
	)

	result := h.run(t, runContext("get_shipment"))

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

	stopped := unsettledToolOutcome("assign_move", temporal.NewCanceledError())
	assert.True(t, stopped.Failed)
	assert.Contains(t, stopped.Content, "stopped before it reported back")
	assert.Contains(t, stopped.Content, "unconfirmed")
	assert.NotContains(t, stopped.Content, "did not happen")
}

// A model call that fails for good ends the turn, and what the turn had done
// comes back with the error, so a write it made is not lost with it.
func TestRunReturnsWhatTheTurnDidWhenTheModelFails(t *testing.T) {
	t.Parallel()

	lookup := &agentruntimetest.StubQueryTool{ToolName: "get_shipment", Result: "found"}
	h := newHarness(t, harnessParams{query: []serviceports.AgentQueryTool{lookup}})
	h.replies(toolReply("get_shipment", map[string]any{"proNumber": "12345"}))
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("bad request", modelcall.ErrTypeModelRejected, nil)).
		Once()

	result := h.run(t, runContext("get_shipment"))

	assert.Contains(t, result.Err, "bad request")
	require.NotNil(t, result.Outcome.Result)
	assert.Equal(t, 1, result.Outcome.Result.ToolCallsUsed)
	assert.Equal(t, 1, lookup.Calls)
}

func TestRunCarriesAProposalBackAcrossTheActivity(t *testing.T) {
	t.Parallel()

	move := &agentruntimetest.StubActionTool{ToolName: "assign_move", Tier: agent.TierPropose}
	h := newHarness(t, harnessParams{action: []serviceports.AgentTool{move}})
	h.replies(
		toolReply("assign_move", map[string]any{"moveId": "mv_1"}),
		textReply("I have proposed the assignment."),
	)

	result := h.run(t, runContext("assign_move"))

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
	settled map[string]serviceports.RunStepOutcome
	claims  int
}

func (l *memoryLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step serviceports.RunStep,
) (serviceports.StepVerdict, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.claims++
	if outcome, ok := l.settled[step.Key]; ok {
		return serviceports.StepVerdict{State: serviceports.StepCompleted, Outcome: outcome}, nil
	}

	return serviceports.StepVerdict{State: serviceports.StepFresh}, nil
}

func (l *memoryLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step serviceports.RunStep,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.settled[step.Key] = step.Outcome

	return nil
}

func (*memoryLedger) Record(context.Context, pagination.TenantInfo, serviceports.RunStep) error {
	return nil
}

func (*memoryLedger) Loaded(
	context.Context,
	pagination.TenantInfo,
	serviceports.RunStepOwner,
) ([]serviceports.RunStep, error) {
	return nil, nil
}

// A write whose activity is lost after the write landed is retried, and the
// retry is told the first attempt's answer instead of writing again.
func TestRunNeverWritesTwiceWhenAToolIsRetried(t *testing.T) {
	t.Parallel()

	move := &agentruntimetest.StubActionTool{ToolName: "assign_move", Tier: agent.TierAutoExecute}
	ledger := &memoryLedger{settled: map[string]serviceports.RunStepOutcome{}}
	h := newHarness(t, harnessParams{
		action: []serviceports.AgentTool{move},
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
		toolReply("assign_move", map[string]any{"moveId": "mv_1"}),
		textReply("Assigned."),
	)

	rc := runContext("assign_move")
	rc.RunID = pulid.MustNew("arun_")
	rc.StepOwner = serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAgentRun, ID: rc.RunID}

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
		{name: "a call to a different tool", activity: "assign_move", call: "get_shipment",
			errType: ErrTypeBadToolInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			move := &agentruntimetest.StubActionTool{ToolName: "assign_move"}
			h := newHarness(t, harnessParams{action: []serviceports.AgentTool{move}})
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
				Run: runContext("assign_move"),
				Call: agentruntime.DispatchCall{
					Call: serviceports.ToolCall{ID: "call_1", Name: tt.call},
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

package agentflow

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replystep"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/runstepledger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	budgets agentbudgetports.AgentBudgetService
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
			Return(&modelstream.Reply{Completion: reply}, nil).Once()
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
			assistantstreamports.AssistantEventMessage,
			assistantstreamports.AssistantEventToolStarted,
			assistantstreamports.AssistantEventToolFinished,
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
	mu       sync.Mutex
	settled  map[string]agentruntimeports.RunStepOutcome
	recorded map[string]agentruntimeports.RunStep
	claims   int
}

func (l *memoryLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step *agentruntimeports.RunStep,
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
	step *agentruntimeports.RunStep,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.settled[step.Key] = step.Outcome

	return nil
}

func (l *memoryLedger) Record(
	_ context.Context,
	_ pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.recorded == nil {
		l.recorded = map[string]agentruntimeports.RunStep{}
	}
	if _, held := l.recorded[step.Key]; !held {
		l.recorded[step.Key] = *step
	}

	return nil
}

func (l *memoryLedger) Find(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentruntimeports.RunStepOwner,
	key string,
) (agentruntimeports.RunStep, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	step, found := l.recorded[key]

	return step, found, nil
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
				Call: toolcall.Call{
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
		Return(func(_ context.Context, in *ModelCallInput) (*modelstream.Reply, error) {
			mu.Lock()
			defer mu.Unlock()
			budgets = append(budgets, in.Request.MaxTokens)

			return &modelstream.Reply{Completion: replies[len(budgets)-1]}, nil
		}).Times(2)

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	assert.Equal(t, []int{0, 2048}, budgets)
	assert.Equal(t, "Nothing is waiting.", result.Outcome.Result.Reply)
	assert.Contains(t, eventNames(result.Outcome), assistantstreamports.AssistantEventRetrying)
}

func TestRunNamesEachModelCallForTheLedger(t *testing.T) {
	t.Parallel()

	lookup := &agenttest.StubQueryTool{ToolName: "shipment_get", Result: "found"}
	h := newHarness(t, harnessParams{query: []agenttoolports.AgentQueryTool{lookup}})
	replies := []*llmports.ChatCompletionResult{
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("It is in Memphis."),
	}
	var (
		mu    sync.Mutex
		steps []*replystep.Call
	)
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelCallInput) (*modelstream.Reply, error) {
			mu.Lock()
			defer mu.Unlock()
			steps = append(steps, in.Step)

			return &modelstream.Reply{Completion: replies[len(steps)-1]}, nil
		}).Times(2)

	rc := runContext("shipment_get")
	rc.RunID = pulid.MustNew("arun_")
	rc.StepOwner = agentruntimeports.RunStepOwner{
		Kind: agentruntimeports.RunStepOwnerAgentRun,
		ID:   rc.RunID,
	}

	result := h.run(t, rc)

	require.Empty(t, result.Err)
	require.Len(t, steps, 2)
	for idx, step := range steps {
		require.NotNil(t, step)
		assert.Equal(t, rc.StepOwner, step.Owner)
		assert.Equal(t, idx+1, step.Ordinal)
		assert.Empty(t, step.Scope)
	}
}

func TestRunNamesNoModelCallForARunWithoutALedger(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessParams{})
	var (
		mu    sync.Mutex
		named []*replystep.Call
	)
	h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.Anything).
		Return(func(_ context.Context, in *ModelCallInput) (*modelstream.Reply, error) {
			mu.Lock()
			defer mu.Unlock()
			named = append(named, in.Step)

			return &modelstream.Reply{Completion: textReply("It is in Memphis.")}, nil
		}).Once()

	result := h.run(t, runContext())

	require.Empty(t, result.Err)
	require.Len(t, named, 1)
	assert.Nil(t, named[0], "a dry run or an evaluation has no ledger to keep its replies in")
}

type stepRows struct {
	mu   sync.Mutex
	rows map[string]*agentrunstep.AgentRunStep
}

func newStepRows() *stepRows {
	return &stepRows{rows: map[string]*agentrunstep.AgentRunStep{}}
}

func (r *stepRows) Claim(
	_ context.Context,
	step *agentrunstep.AgentRunStep,
) (*agentrunstep.AgentRunStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if held, ok := r.rows[step.StepKey]; ok {
		return held, agentruntimeports.ErrRunStepClaimed
	}
	r.rows[step.StepKey] = step

	return step, nil
}

func (r *stepRows) Settle(
	_ context.Context,
	req *agentruntimeports.SettleAgentRunStepRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if held, ok := r.rows[req.StepKey]; ok {
		held.Status = req.Status
		held.Outcome = req.Outcome
	}

	return nil
}

func (r *stepRows) Record(_ context.Context, step *agentrunstep.AgentRunStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, held := r.rows[step.StepKey]; !held {
		r.rows[step.StepKey] = step
	}

	return nil
}

func (r *stepRows) List(
	context.Context,
	*agentruntimeports.ListAgentRunStepsRequest,
) ([]*agentrunstep.AgentRunStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows := make([]*agentrunstep.AgentRunStep, 0, len(r.rows))
	for _, row := range r.rows {
		rows = append(rows, row)
	}

	return rows, nil
}

func (r *stepRows) Get(
	_ context.Context,
	req *agentruntimeports.GetAgentRunStepRequest,
) (*agentrunstep.AgentRunStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	row, ok := r.rows[req.StepKey]
	if !ok || row.OwnerID != req.OwnerID {
		return nil, errortypes.NewNotFoundError("AgentRunStep not found within your organization")
	}

	return row, nil
}

func (*stepRows) Prune(context.Context, agentruntimeports.PruneAgentRunStepsRequest) (int, error) {
	return 0, nil
}

func (r *stepRows) completions() []*agentrunstep.AgentRunStep {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := make([]*agentrunstep.AgentRunStep, 0, len(r.rows))
	for _, row := range r.rows {
		if row.Kind == string(agentruntimeports.RunStepCompletion) {
			kept = append(kept, row)
		}
	}

	return kept
}

func modelCallActivities(
	t *testing.T,
	rows *stepRows,
	completion *agenttest.ScriptedCompletion,
) (*Activities, *testsuite.TestActivityEnvironment) {
	t.Helper()

	rt := agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  completion,
		QueryTools:  &agenttest.StubQueryRegistry{},
		ActionTools: &agenttest.StubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
	})
	acts := NewActivities(ActivitiesParams{
		Logger:  zap.NewNop(),
		Runtime: rt,
		Steps:   runstepledger.New(runstepledger.Params{Logger: zap.NewNop(), Repo: rows}),
	})

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(acts)

	return acts, env
}

func modelCallInput() *ModelCallInput {
	return &ModelCallInput{
		Request: &llmports.ChatCompletionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			System:   "Help dispatch.",
			Messages: llmports.UserMessage("Where is load 12345?"),
		},
		Step: &replystep.Call{
			Owner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAssistantTurn,
				ID:   pulid.MustNew("atrn_"),
			},
			Ordinal: 1,
		},
	}
}

func callModel(
	t *testing.T,
	acts *Activities,
	env *testsuite.TestActivityEnvironment,
	in *ModelCallInput,
) *modelstream.Reply {
	t.Helper()

	encoded, err := env.ExecuteActivity(acts.ModelCallActivity, in)
	require.NoError(t, err)

	var reply modelstream.Reply
	require.NoError(t, encoded.Get(&reply))

	return &reply
}

func TestModelCallKeepsEachReplyInTheLedger(t *testing.T) {
	t.Parallel()

	rows := newStepRows()
	completion := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolReply("shipment_get", map[string]any{"proNumber": "12345"}),
		textReply("Load 12345 is in Memphis."),
	}}
	acts, env := modelCallActivities(t, rows, completion)

	first := modelCallInput()
	second := modelCallInput()
	second.Step.Owner = first.Step.Owner
	second.Step.Ordinal = 2
	second.Request.TenantInfo = first.Request.TenantInfo

	callModel(t, acts, env, first)
	callModel(t, acts, env, second)

	assert.Equal(t, 2, completion.CallCount)
	kept := rows.completions()
	require.Len(t, kept, 2, "each reply is a step of its own")
	assert.NotEqual(t, kept[0].StepKey, kept[1].StepKey)
	for _, row := range kept {
		assert.Equal(t, string(agentruntimeports.RunStepCompleted), row.Status)
		assert.Equal(t, string(agentruntimeports.RunStepOwnerAssistantTurn), row.OwnerKind)
		assert.Equal(t, first.Step.Owner.ID, row.OwnerID)
		assert.Equal(t, first.Request.TenantInfo.OrgID, row.OrganizationID)
		assert.Equal(t, 1, row.Attempt)
		assert.True(t, strings.HasPrefix(row.StepKey, "completion:"))
		assert.NotNil(t, row.Outcome["reply"], "the whole reply is kept")
		assert.NotNil(t, row.SettledAtMs)
	}
}

func TestModelCallReplaysAReplyAnEarlierAttemptKept(t *testing.T) {
	t.Parallel()

	rows := newStepRows()
	in := modelCallInput()
	answered := toolReply("shipment_get", map[string]any{"proNumber": "12345"})
	answered.Text = "Looking it up."
	answered.InputTokens = 1200
	answered.OutputTokens = 40
	first := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{answered}}
	acts, env := modelCallActivities(t, rows, first)
	original := callModel(t, acts, env, in)

	resumed := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		textReply("A different answer."),
	}}
	acts, env = modelCallActivities(t, rows, resumed)
	replayed := callModel(t, acts, env, in)

	assert.Equal(t, 1, first.CallCount)
	assert.Zero(t, resumed.CallCount, "a resumed run does not ask the model again")
	require.NotNil(t, replayed.Completion)
	assert.Equal(t, "Looking it up.", replayed.Completion.Text)
	assert.Equal(t, "test-model", replayed.Completion.ModelIdentifier)
	require.Len(t, replayed.Completion.ToolCalls, 1)
	assert.Equal(t, "shipment_get", replayed.Completion.ToolCalls[0].Name)
	assert.Equal(t, "12345", replayed.Completion.ToolCalls[0].Arguments["proNumber"])
	assert.Equal(t, original.TotalUsage(), replayed.TotalUsage())
	assert.Len(t, rows.completions(), 1)
}

func TestModelCallAsksAgainWhenTheRequestChanged(t *testing.T) {
	t.Parallel()

	rows := newStepRows()
	in := modelCallInput()
	first := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		textReply("Load 12345 is in Memphis."),
	}}
	acts, env := modelCallActivities(t, rows, first)
	callModel(t, acts, env, in)

	request := *in.Request
	request.Messages = llmports.UserMessage("Where is load 67890?")
	changed := &ModelCallInput{Request: &request, Step: in.Step}
	resumed := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		textReply("Load 67890 is in Dallas."),
	}}
	acts, env = modelCallActivities(t, rows, resumed)
	reply := callModel(t, acts, env, changed)

	assert.Equal(t, 1, resumed.CallCount, "a reply answers only the question it was given")
	assert.Equal(t, "Load 67890 is in Dallas.", reply.Completion.Text)
	assert.Len(t, rows.completions(), 2)
}

func TestModelCallAsksAgainWhenTheReplyWasTooLargeToKeep(t *testing.T) {
	t.Parallel()

	rows := newStepRows()
	in := modelCallInput()
	var text strings.Builder
	for idx := range 3_000 {
		text.WriteString("Shipment " + strconv.Itoa(idx) + " is in transit.\n")
	}
	large := textReply(text.String())
	first := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{large}}
	acts, env := modelCallActivities(t, rows, first)
	callModel(t, acts, env, in)

	kept := rows.completions()
	require.Len(t, kept, 1, "the step is still on record")
	assert.Nil(t, kept[0].Outcome["reply"], "without the reply it could not hold")

	resumed := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		textReply("Load 12345 is in Memphis."),
	}}
	acts, env = modelCallActivities(t, rows, resumed)
	reply := callModel(t, acts, env, in)

	assert.Equal(t, 1, resumed.CallCount, "a reply that was not kept is asked for again")
	assert.Equal(t, "Load 12345 is in Memphis.", reply.Completion.Text)
	assert.Len(t, rows.completions(), 1, "the step keeps the row it was first filed under")
}

func TestModelCallKeepsNoReplyForAFailedCall(t *testing.T) {
	t.Parallel()

	rows := newStepRows()
	completion := &agenttest.ScriptedCompletion{
		Turns:  []*llmports.ChatCompletionResult{textReply("unused")},
		Errors: map[int]error{0: errors.New("provider unavailable")},
	}
	acts, env := modelCallActivities(t, rows, completion)

	_, err := env.ExecuteActivity(acts.ModelCallActivity, modelCallInput())

	require.Error(t, err)
	assert.Empty(t, rows.completions(), "a call that failed left no reply to replay")
}

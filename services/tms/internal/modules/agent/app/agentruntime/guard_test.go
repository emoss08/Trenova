package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLedger stands in for the step ledger. Verdict is what every claim
// returns; Claimed and Settled record what the runtime asked for.
type stubLedger struct {
	Verdict agentports.StepVerdict
	Err     error
	Loads   []agentports.RunStep

	Claimed []agentports.RunStep
	Settled []agentports.RunStep
}

func (l *stubLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentports.RunStep,
) (agentports.StepVerdict, error) {
	l.Claimed = append(l.Claimed, step)
	if l.Err != nil {
		return agentports.StepVerdict{}, l.Err
	}

	return l.Verdict, nil
}

func (l *stubLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentports.RunStep,
) error {
	l.Settled = append(l.Settled, step)

	return nil
}

func (l *stubLedger) Record(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentports.RunStep,
) error {
	return nil
}

func (l *stubLedger) Loaded(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentports.RunStepOwner,
) ([]agentports.RunStep, error) {
	return l.Loads, nil
}

// guardedRun wires an auto-executing write tool to a ledger and runs one turn
// that calls it.
func guardedRun(
	t *testing.T,
	ledger *stubLedger,
	toolErr error,
) (*agentports.RunResult, *agenttest.StubActionTool) {
	t.Helper()

	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, toolErr)
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{action}}, nil)

	definition := testDefinition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{"dispatch_assign_move": agent.TierAutoExecute}

	runID := pulid.MustNew("ar_")
	result, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Assign mv_1",
		RunID:      runID,
		Steps:      ledger,
		StepOwner: agentports.RunStepOwner{
			Kind: agentports.RunStepOwnerAgentRun,
			ID:   runID,
		},
		Attempt: 2,
	})
	require.NoError(t, err)

	return result, action
}

func TestGuardedDispatch_RunsAFreshCallAndSettlesIt(t *testing.T) {
	t.Parallel()

	ledger := &stubLedger{Verdict: agentports.StepVerdict{State: agentports.StepFresh}}
	result, action := guardedRun(t, ledger, nil)

	require.Equal(t, 1, action.Calls)
	require.Len(t, ledger.Claimed, 1)
	assert.Equal(t, agentports.RunStepTool, ledger.Claimed[0].Kind)
	assert.Equal(t, 2, ledger.Claimed[0].Attempt)

	// The tool is handed the step key, not the provider's call id: that is
	// what makes a key-aware tool refuse the same write on the next attempt.
	assert.NotEqual(t, "call_1", action.LastParams.IdempotencyKey)
	assert.Equal(t, ledger.Claimed[0].Key, action.LastParams.IdempotencyKey)

	require.Len(t, ledger.Settled, 1)
	assert.Equal(t, agentports.RunStepCompleted, ledger.Settled[0].Status)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}

func TestGuardedDispatch_DoesNotRunAWriteAnEarlierAttemptAlreadyMade(t *testing.T) {
	t.Parallel()

	// This is the bug the ledger exists for. Before it, a retried activity
	// re-ran every write the first attempt had made, and the declared
	// idempotency key did not help because nothing looks it up.
	pinned := &agentports.PendingAction{
		ToolName:  "dispatch_assign_move",
		Arguments: map[string]any{"moveId": "mv_1"},
		Executed:  true,
		Target: &agentports.ProposalTarget{
			ID:      pulid.MustNew("shp_"),
			Version: 7,
		},
	}
	ledger := &stubLedger{Verdict: agentports.StepVerdict{
		State: agentports.StepCompleted,
		Outcome: agentports.RunStepOutcome{
			Content: `Tool "dispatch_assign_move" ran successfully.`,
			Action:  pinned,
		},
	}}

	result, action := guardedRun(t, ledger, nil)

	assert.Zero(t, action.Calls, "the write had already been made")
	assert.Empty(t, ledger.Settled, "a replayed step is not settled again")

	require.Len(t, result.Actions, 1)
	assert.Equal(t, `Tool "dispatch_assign_move" ran successfully.`, result.Messages[2].Content)

	// The pinned version comes back as it was taken. Re-pinning it here would
	// record the record as it stands now and quietly defeat the staleness
	// check the pin exists for.
	require.NotNil(t, result.Actions[0].Target)
	assert.Equal(t, int64(7), result.Actions[0].Target.Version)
}

func TestGuardedDispatch_TellsTheModelWhenAnEarlierAttemptLeftAWriteUnconfirmed(t *testing.T) {
	t.Parallel()

	ledger := &stubLedger{Verdict: agentports.StepVerdict{State: agentports.StepUnknown}}
	result, action := guardedRun(t, ledger, nil)

	assert.Zero(t, action.Calls, "nobody knows whether this ran; it is not run again")
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "unconfirmed")
	assert.Empty(t, result.Actions, "nothing is claimed to have happened")
}

func TestGuardedDispatch_RefusesTheCallWhenTheLedgerCannotBeWritten(t *testing.T) {
	t.Parallel()

	// Running unguarded here would produce the duplicate the ledger exists to
	// prevent, and would do it exactly when the database is already unwell.
	ledger := &stubLedger{Err: errors.New("connection refused")}
	result, action := guardedRun(t, ledger, nil)

	assert.Zero(t, action.Calls)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "could not record")
}

func TestGuardedDispatch_SettlesAFailedCallAsFailed(t *testing.T) {
	t.Parallel()

	ledger := &stubLedger{Verdict: agentports.StepVerdict{State: agentports.StepFresh}}
	_, action := guardedRun(t, ledger, errors.New("driver is out of hours"))

	require.Equal(t, 1, action.Calls)
	require.Len(t, ledger.Settled, 1)
	assert.Equal(t, agentports.RunStepFailed, ledger.Settled[0].Status)
	assert.True(t, ledger.Settled[0].Outcome.Failed)
}

func TestSeedFromLedger_RefusesACallAnEarlierAttemptAlreadyFailedOn(t *testing.T) {
	t.Parallel()

	// The repeat guard is per-attempt and in memory, so a retry starts blank
	// and would spend budget re-learning a refusal the first attempt already
	// had.
	args := map[string]any{"moveId": "mv_1"}
	ledger := &stubLedger{
		Verdict: agentports.StepVerdict{State: agentports.StepFresh},
		Loads: []agentports.RunStep{{
			Kind:     agentports.RunStepTool,
			Status:   agentports.RunStepFailed,
			ToolName: "dispatch_assign_move",
			Args:     args,
			Outcome: agentports.RunStepOutcome{
				Content: "driver is out of hours",
				Failed:  true,
			},
		}},
	}

	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("dispatch_assign_move", args),
		textTurn("I could not assign it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{action}}, nil)
	definition := testDefinition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{"dispatch_assign_move": agent.TierAutoExecute}

	runID := pulid.MustNew("ar_")
	result, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Assign mv_1",
		RunID:      runID,
		Steps:      ledger,
		StepOwner:  agentports.RunStepOwner{Kind: agentports.RunStepOwnerAgentRun, ID: runID},
		Attempt:    2,
	})
	require.NoError(t, err)

	assert.Zero(t, action.Calls, "the identical call already failed on the first attempt")
	assert.Empty(t, ledger.Claimed, "a refused repeat never reaches the ledger")
	assert.Contains(t, result.Messages[2].Content, "driver is out of hours")
}

func TestRun_LeavesAnUnguardedRunOnTheProvidersCallID(t *testing.T) {
	t.Parallel()

	// A run with no ledger behaves exactly as it did before this existed.
	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{action}}, nil)
	definition := testDefinition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{"dispatch_assign_move": agent.TierAutoExecute}

	_, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Assign mv_1",
	})
	require.NoError(t, err)

	require.Equal(t, 1, action.Calls)
	assert.Equal(t, "call_1", action.LastParams.IdempotencyKey)
}

// keyedLedger answers by key, the way the real ledger does: a key it has
// settled is replayed, and any other key is fresh. The fixed-verdict stub
// above answers the same for every key, which is how a retry that minted a
// new key for an old write passed.
type keyedLedger struct {
	steps map[string]agentports.RunStep
}

func newKeyedLedger() *keyedLedger {
	return &keyedLedger{steps: map[string]agentports.RunStep{}}
}

func (l *keyedLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentports.RunStep,
) (agentports.StepVerdict, error) {
	recorded, seen := l.steps[step.Key]
	switch {
	case !seen:
		l.steps[step.Key] = step
		return agentports.StepVerdict{State: agentports.StepFresh}, nil
	case recorded.Status == agentports.RunStepCompleted:
		return agentports.StepVerdict{State: agentports.StepCompleted, Outcome: recorded.Outcome}, nil
	case recorded.Status == agentports.RunStepFailed:
		return agentports.StepVerdict{State: agentports.StepFailed, Outcome: recorded.Outcome}, nil
	default:
		return agentports.StepVerdict{State: agentports.StepUnknown}, nil
	}
}

func (l *keyedLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentports.RunStep,
) error {
	l.steps[step.Key] = step

	return nil
}

func (l *keyedLedger) Record(context.Context, pagination.TenantInfo, agentports.RunStep) error {
	return nil
}

func (l *keyedLedger) Loaded(
	context.Context,
	pagination.TenantInfo,
	agentports.RunStepOwner,
) ([]agentports.RunStep, error) {
	steps := make([]agentports.RunStep, 0, len(l.steps))
	for _, step := range l.steps {
		steps = append(steps, step)
	}

	return steps, nil
}

/*
An attempt that starts over asks the model afresh, and the model asks for the
same write again. It has to reach the same step key as the first attempt did,
so the ledger answers with what already happened.

Numbering the retry's calls after the ones on record minted a new key for the
same write, and a move was assigned twice.
*/
func TestRun_ARetriedAttemptDoesNotMakeTheSameWriteTwice(t *testing.T) {
	t.Parallel()

	ledger := newKeyedLedger()
	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	runID := pulid.MustNew("ar_")
	definition := testDefinition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{"dispatch_assign_move": agent.TierAutoExecute}

	for attempt := 1; attempt <= 2; attempt++ {
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
			textTurn("Assigned."),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{},
			&stubActionRegistry{Tools: []agentports.AgentTool{action}}, nil)

		_, err := rt.Run(t.Context(), &agentports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      "Assign mv_1",
			RunID:      runID,
			Steps:      ledger,
			StepOwner:  agentports.RunStepOwner{Kind: agentports.RunStepOwnerAgentRun, ID: runID},
			Attempt:    attempt,
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, action.Calls, "the second attempt replays the first attempt's write")
}

/*
A step an earlier attempt already ran is shown again beside the conversation.

The replay handed back what the model had been told and nothing else, so the
observer was given no result, built no artifact, and the table or card the
original call had put in the pane was missing from the reply that followed.
*/
func TestRun_AReplayedStepShowsWhatTheOriginalCallShowed(t *testing.T) {
	t.Parallel()

	ledger := newKeyedLedger()
	search := queryTool("worker_list", map[string]any{
		"results": []any{map[string]any{"name": "Maria Ortiz"}},
	}, nil)
	runID := pulid.MustNew("ar_")
	threadID := pulid.MustNew("th_")

	shown := make([]any, 0, 2)
	for attempt := 1; attempt <= 2; attempt++ {
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("worker_list", map[string]any{"query": "Maria"}),
			textTurn("Maria Ortiz."),
		}}
		rt := newRuntime(completion,
			&stubQueryRegistry{Tools: []agentports.AgentQueryTool{search}},
			&stubActionRegistry{}, nil)

		_, err := rt.Run(t.Context(), &agentports.RunRequest{
			Definition: testDefinition("worker_list"),
			Actor:      testActor(),
			Input:      "Who is Maria?",
			RunID:      runID,
			ThreadID:   threadID,
			Steps:      ledger,
			StepOwner:  agentports.RunStepOwner{Kind: agentports.RunStepOwnerAgentRun, ID: runID},
			Attempt:    attempt,
			ToolObserver: func(
				observation agentports.ToolObservation,
			) (*agentports.ShownArtifact, error) {
				shown = append(shown, observation.Data)
				return nil, nil
			},
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, search.Calls, "the second attempt replays the first attempt's call")
	require.Len(t, shown, 2)
	require.NotNil(t, shown[1], "the replayed call is shown its result")
	replayed, ok := shown[1].(map[string]any)
	require.True(t, ok)
	results, ok := replayed["results"].([]any)
	require.True(t, ok)
	require.Len(t, results, 1)
	assert.Equal(t, "Maria Ortiz", results[0].(map[string]any)["name"])
}

func TestStepData_KeepsOnlyAnObjectSmallEnoughToShowAgain(t *testing.T) {
	t.Parallel()

	assert.Nil(t, stepData(nil))
	assert.Nil(t, stepData([]any{"a"}), "a bare list builds no artifact")
	assert.Nil(t, stepData(map[string]any{"cycle": make(chan int)}))
	assert.Nil(t, stepData(map[string]any{"blob": string(make([]byte, maxStoredStepDataBytes))}),
		"a result past the bound is dropped, not truncated")
	assert.Equal(t, map[string]any{"count": float64(2)}, stepData(map[string]any{"count": 2}))
}

var _ agentports.RunStepLedger = (*keyedLedger)(nil)

func TestRun_TheObserverIsToldTheArgumentsTheToolRanWith(t *testing.T) {
	t.Parallel()

	ledger := newKeyedLedger()
	list := queryTool("shipment_list", map[string]any{
		"items": []any{map[string]any{"proNumber": "S-1001"}},
	}, nil)
	list.Schema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
	runID := pulid.MustNew("ar_")
	threadID := pulid.MustNew("th_")

	ranWith := make([]map[string]any, 0, 2)
	for attempt := 1; attempt <= 2; attempt++ {
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("shipment_list", map[string]any{"ids": "shp_1"}),
			textTurn("One shipment."),
		}}
		rt := newRuntime(completion,
			&stubQueryRegistry{Tools: []agentports.AgentQueryTool{list}},
			&stubActionRegistry{}, nil)

		_, err := rt.Run(t.Context(), &agentports.RunRequest{
			Definition: testDefinition("shipment_list"),
			Actor:      testActor(),
			Input:      "Show shipment shp_1",
			RunID:      runID,
			ThreadID:   threadID,
			Steps:      ledger,
			StepOwner:  agentports.RunStepOwner{Kind: agentports.RunStepOwnerAgentRun, ID: runID},
			Attempt:    attempt,
			ToolObserver: func(
				observation agentports.ToolObservation,
			) (*agentports.ShownArtifact, error) {
				assert.Equal(t, "shp_1", observation.Call.Arguments["ids"])
				ranWith = append(ranWith, observation.RanWith)
				return nil, nil
			},
		})
		require.NoError(t, err)
	}

	want := map[string]any{"ids": []any{"shp_1"}}
	assert.Equal(t, 1, list.Calls)
	assert.Equal(t, want, list.LastParams.Params)
	assert.Equal(t, []map[string]any{want, want}, ranWith)
}

package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLedger stands in for the step ledger. Verdict is what every claim
// returns; Claimed and Settled record what the runtime asked for.
type stubLedger struct {
	Verdict agentruntimeports.StepVerdict
	Err     error
	Loads   []agentruntimeports.RunStep

	Claimed []agentruntimeports.RunStep
	Settled []agentruntimeports.RunStep
}

func (l *stubLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentruntimeports.RunStep,
) (agentruntimeports.StepVerdict, error) {
	l.Claimed = append(l.Claimed, step)
	if l.Err != nil {
		return agentruntimeports.StepVerdict{}, l.Err
	}

	return l.Verdict, nil
}

func (l *stubLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step agentruntimeports.RunStep,
) error {
	l.Settled = append(l.Settled, step)

	return nil
}

func (l *stubLedger) Record(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentruntimeports.RunStep,
) error {
	return nil
}

func (l *stubLedger) Loaded(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentruntimeports.RunStepOwner,
) ([]agentruntimeports.RunStep, error) {
	return l.Loads, nil
}

// guardedRun wires an auto-executing write tool to a ledger and runs one turn
// that calls it.
func guardedRun(
	t *testing.T,
	ledger *stubLedger,
	toolErr error,
) (*agentruntimeports.RunResult, *agenttest.StubActionTool) {
	t.Helper()

	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, toolErr)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)

	definition := runtimetest.Definition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{
		"dispatch_assign_move": agent.TierAutoExecute,
	}

	runID := pulid.MustNew("ar_")
	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Assign mv_1",
		RunID:      runID,
		Steps:      ledger,
		StepOwner: agentruntimeports.RunStepOwner{
			Kind: agentruntimeports.RunStepOwnerAgentRun,
			ID:   runID,
		},
		Attempt: 2,
	})
	require.NoError(t, err)

	return result, action
}

func TestGuardedDispatch_RunsAFreshCallAndSettlesIt(t *testing.T) {
	t.Parallel()

	ledger := &stubLedger{
		Verdict: agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh},
	}
	result, action := guardedRun(t, ledger, nil)

	require.Equal(t, 1, action.Calls)
	require.Len(t, ledger.Claimed, 1)
	assert.Equal(t, agentruntimeports.RunStepTool, ledger.Claimed[0].Kind)
	assert.Equal(t, 2, ledger.Claimed[0].Attempt)

	// The tool is handed the step key, not the provider's call id: that is
	// what makes a key-aware tool refuse the same write on the next attempt.
	assert.NotEqual(t, "call_1", action.LastParams.IdempotencyKey)
	assert.Equal(t, ledger.Claimed[0].Key, action.LastParams.IdempotencyKey)

	require.Len(t, ledger.Settled, 1)
	assert.Equal(t, agentruntimeports.RunStepCompleted, ledger.Settled[0].Status)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}

func TestGuardedDispatch_DoesNotRunAWriteAnEarlierAttemptAlreadyMade(t *testing.T) {
	t.Parallel()

	// This is the bug the ledger exists for. Before it, a retried activity
	// re-ran every write the first attempt had made, and the declared
	// idempotency key did not help because nothing looks it up.
	pinned := &agentruntimeports.PendingAction{
		ToolName:  "dispatch_assign_move",
		Arguments: map[string]any{"moveId": "mv_1"},
		Executed:  true,
		Target: &agentruntimeports.ProposalTarget{
			ID:      pulid.MustNew("shp_"),
			Version: 7,
		},
	}
	ledger := &stubLedger{Verdict: agentruntimeports.StepVerdict{
		State: agentruntimeports.StepCompleted,
		Outcome: agentruntimeports.RunStepOutcome{
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

	ledger := &stubLedger{
		Verdict: agentruntimeports.StepVerdict{State: agentruntimeports.StepUnknown},
	}
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

	ledger := &stubLedger{
		Verdict: agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh},
	}
	_, action := guardedRun(t, ledger, errors.New("driver is out of hours"))

	require.Equal(t, 1, action.Calls)
	require.Len(t, ledger.Settled, 1)
	assert.Equal(t, agentruntimeports.RunStepFailed, ledger.Settled[0].Status)
	assert.True(t, ledger.Settled[0].Outcome.Failed)
}

func TestSeedFromLedger_RefusesACallAnEarlierAttemptAlreadyFailedOn(t *testing.T) {
	t.Parallel()

	// The repeat guard is per-attempt and in memory, so a retry starts blank
	// and would spend budget re-learning a refusal the first attempt already
	// had.
	args := map[string]any{"moveId": "mv_1"}
	ledger := &stubLedger{
		Verdict: agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh},
		Loads: []agentruntimeports.RunStep{{
			Kind:     agentruntimeports.RunStepTool,
			Status:   agentruntimeports.RunStepFailed,
			ToolName: "dispatch_assign_move",
			Args:     args,
			Outcome: agentruntimeports.RunStepOutcome{
				Content: "driver is out of hours",
				Failed:  true,
			},
		}},
	}

	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", args),
		runtimetest.TextTurn("I could not assign it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := runtimetest.Definition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{
		"dispatch_assign_move": agent.TierAutoExecute,
	}

	runID := pulid.MustNew("ar_")
	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Assign mv_1",
		RunID:      runID,
		Steps:      ledger,
		StepOwner: agentruntimeports.RunStepOwner{
			Kind: agentruntimeports.RunStepOwnerAgentRun,
			ID:   runID,
		},
		Attempt: 2,
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
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)
	definition := runtimetest.Definition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{
		"dispatch_assign_move": agent.TierAutoExecute,
	}

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Assign mv_1",
	})
	require.NoError(t, err)

	require.Equal(t, 1, action.Calls)
	assert.Equal(t, "call_1", action.LastParams.IdempotencyKey)
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

	ledger := runtimetest.NewKeyedLedger()
	action := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	runID := pulid.MustNew("ar_")
	definition := runtimetest.Definition("dispatch_assign_move")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{
		"dispatch_assign_move": agent.TierAutoExecute,
	}

	for attempt := 1; attempt <= 2; attempt++ {
		completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
			runtimetest.TextTurn("Assigned."),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{},
			&stubActionRegistry{Tools: []agenttoolports.AgentTool{action}}, nil)

		_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
			Definition: definition,
			Actor:      runtimetest.Actor(),
			Input:      "Assign mv_1",
			RunID:      runID,
			Steps:      ledger,
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAgentRun,
				ID:   runID,
			},
			Attempt: attempt,
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

	ledger := runtimetest.NewKeyedLedger()
	search := queryTool("worker_list", map[string]any{
		"results": []any{map[string]any{"name": "Maria Ortiz"}},
	}, nil)
	runID := pulid.MustNew("ar_")
	threadID := pulid.MustNew("th_")

	shown := make([]any, 0, 2)
	for attempt := 1; attempt <= 2; attempt++ {
		completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn("worker_list", map[string]any{"query": "Maria"}),
			runtimetest.TextTurn("Maria Ortiz."),
		}}
		rt := newRuntime(completion,
			&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{search}},
			&stubActionRegistry{}, nil)

		_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition("worker_list"),
			Actor:      runtimetest.Actor(),
			Input:      "Who is Maria?",
			RunID:      runID,
			ThreadID:   threadID,
			Steps:      ledger,
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAgentRun,
				ID:   runID,
			},
			Attempt: attempt,
			ToolObserver: func(
				observation agentruntimeports.ToolObservation,
			) (*agentruntimeports.ShownArtifact, error) {
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

var _ agentruntimeports.RunStepLedger = (*runtimetest.KeyedLedger)(nil)

func TestRun_TheObserverIsToldTheArgumentsTheToolRanWith(t *testing.T) {
	t.Parallel()

	ledger := runtimetest.NewKeyedLedger()
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
		completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn("shipment_list", map[string]any{"ids": "shp_1"}),
			runtimetest.TextTurn("One shipment."),
		}}
		rt := newRuntime(completion,
			&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{list}},
			&stubActionRegistry{}, nil)

		_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition("shipment_list"),
			Actor:      runtimetest.Actor(),
			Input:      "Show shipment shp_1",
			RunID:      runID,
			ThreadID:   threadID,
			Steps:      ledger,
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAgentRun,
				ID:   runID,
			},
			Attempt: attempt,
			ToolObserver: func(
				observation agentruntimeports.ToolObservation,
			) (*agentruntimeports.ShownArtifact, error) {
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

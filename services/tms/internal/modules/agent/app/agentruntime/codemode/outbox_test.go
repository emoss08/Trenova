package codemode_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptStep = "script-step"

func intent(tool string, id string) (string, map[string]any, *agentports.PendingAction) {
	args := map[string]any{"invoiceIds": []any{id}}

	return tool, args, &agentports.PendingAction{
		ToolName:  tool,
		Arguments: args,
		Tier:      agent.TierActWithApproval,
		Egress:    agent.EgressMoney,
		HeldBy:    []string{agenttoolpolicy.HeldByScriptIntent},
	}
}

func hold(t *testing.T, outbox *codemode.Outbox, tool string, id string) int {
	t.Helper()

	name, args, action := intent(tool, id)
	key, err := outbox.Admit(name, args)
	require.NoError(t, err)
	seq, err := outbox.Hold(key, action)
	require.NoError(t, err)

	return seq
}

func completed() codemode.Settlement {
	return codemode.Settlement{Completed: true}
}

func TestOutbox_FilesEveryIntentOnceInTheOrderPrepared(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	for idx := range 7 {
		assert.Equal(t, idx+1, hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx)))
	}

	filed := outbox.Settle(completed())
	require.Len(t, filed, 7)
	for idx, action := range filed {
		assert.Equal(t, []any{fmt.Sprintf("inv_%d", idx)}, action.Arguments["invoiceIds"])
		assert.Equal(t, scriptStep, action.ScriptStepKey)
		assert.False(t, action.Executed)
	}
	assert.Nil(t, outbox.Settle(completed()), "a batch is filed once")
}

func TestOutbox_FilesNothingForAScriptThatDidNotFinish(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")
	hold(t, outbox, "invoice_post", "inv_2")

	assert.Empty(t, outbox.Settle(codemode.Settlement{Completed: false}))
	_, err := outbox.Admit("invoice_post", map[string]any{"invoiceIds": []any{"inv_3"}})
	require.ErrorIs(t, err, codemode.ErrOutboxFiled)
}

func TestOutbox_RefusesADuplicateAndKeepsTheFirst(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	for range 2 {
		_, err := outbox.Admit("invoice_post", map[string]any{"invoiceIds": []any{"inv_1"}})
		require.ErrorIs(t, err, codemode.ErrDuplicateIntent)
		assert.Contains(t, err.Error(), "change 1")
	}
	_, err := outbox.Admit("invoice_send", map[string]any{"invoiceIds": []any{"inv_1"}})
	require.NoError(t, err, "another tool on the same record is another change")

	assert.Len(t, outbox.Settle(completed()), 1)
}

func TestOutbox_RefusesAnIntentPastTheBatchsSize(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 0)
	for idx := range codemode.MaxBatchSteps {
		hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx))
	}

	_, err := outbox.Admit("invoice_post", map[string]any{"invoiceIds": []any{"inv_over"}})
	require.ErrorIs(t, err, codemode.ErrBatchTooLarge)
	assert.Len(t, outbox.Settle(completed()), codemode.MaxBatchSteps)
}

func TestOutbox_HoldsTheAgentsChangeLimitWhenItIsSmaller(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 3)
	for idx := range 3 {
		hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx))
	}

	_, err := outbox.Admit("invoice_post", map[string]any{"invoiceIds": []any{"inv_over"}})
	require.ErrorIs(t, err, codemode.ErrBatchTooLarge)
}

func TestOutbox_RefusesACallThatRanOrWouldRun(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(*agentports.PendingAction){
		"executed":  func(a *agentports.PendingAction) { a.Executed = true },
		"simulated": func(a *agentports.PendingAction) { a.Simulated = true },
		"automatic": func(a *agentports.PendingAction) { a.Tier = agent.TierAutoExecute },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outbox := codemode.NewOutbox(scriptStep, 500)
			tool, args, action := intent("invoice_post", "inv_1")
			mutate(action)
			key, err := outbox.Admit(tool, args)
			require.NoError(t, err)
			_, err = outbox.Hold(key, action)
			require.ErrorIs(t, err, codemode.ErrIntentRan)
			assert.Zero(t, outbox.Len())
		})
	}
}

func TestOutbox_HoldsACopyTheCallerCannotChange(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	tool, args, action := intent("invoice_post", "inv_1")
	key, err := outbox.Admit(tool, args)
	require.NoError(t, err)
	_, err = outbox.Hold(key, action)
	require.NoError(t, err)
	args["invoiceIds"] = []any{"inv_other"}
	action.Tier = agent.TierAutoExecute

	filed := outbox.Settle(completed())
	require.Len(t, filed, 1)
	assert.Equal(t, []any{"inv_1"}, filed[0].Arguments["invoiceIds"])
	assert.Equal(t, agent.TierActWithApproval, filed[0].Tier)
}

/*
The script's statements run in an order the host cannot see, so a read after
an intent still holds it: every intent is decided again at filing with the
taint of the whole script.
*/
func TestOutbox_DecidesEveryIntentAgainWithTheScriptsWholeTaint(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")
	hold(t, outbox, "invoice_post", "inv_2")

	redecided := 0
	filed := outbox.Settle(codemode.Settlement{
		Completed: true,
		Tainted:   true,
		Redecide: func(*agentports.PendingAction) agenttoolpolicy.Decision {
			redecided++

			return agenttoolpolicy.Decision{
				Tier:   agent.TierPropose,
				HeldBy: []string{agenttoolpolicy.HeldByTainted, agenttoolpolicy.HeldByScriptIntent},
			}
		},
	})

	assert.Equal(t, 2, redecided)
	for _, action := range filed {
		assert.True(t, action.Tainted)
		assert.Equal(t, agent.TierPropose, action.Tier)
		assert.Equal(
			t,
			[]string{agenttoolpolicy.HeldByScriptIntent, agenttoolpolicy.HeldByTainted},
			action.HeldBy,
		)
	}
}

func TestOutbox_CapsEveryIntentAtProposeAfterOutsideContent(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	filed := outbox.Settle(codemode.Settlement{Completed: true, ExternalContent: true})
	require.Len(t, filed, 1)
	assert.Equal(t, agent.TierPropose, filed[0].Tier)
	assert.True(t, filed[0].Tainted)
	assert.Contains(t, filed[0].HeldBy, agenttoolpolicy.HeldByTainted)
}

func TestOutbox_NeverFilesAnIntentThatRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	filed := outbox.Settle(codemode.Settlement{
		Completed: true,
		Redecide: func(*agentports.PendingAction) agenttoolpolicy.Decision {
			return agenttoolpolicy.Decision{Tier: agent.TierAutoExecute}
		},
	})
	require.Len(t, filed, 1)
	assert.Equal(t, agent.TierActWithApproval, filed[0].Tier, "a decision can only lower an intent")
	assert.Contains(t, filed[0].HeldBy, agenttoolpolicy.HeldByScriptIntent)
}

func TestOutbox_AdmitsConcurrentCallsWithoutLosingOrRepeatingOne(t *testing.T) {
	t.Parallel()

	outbox := codemode.NewOutbox(scriptStep, 500)
	var wg sync.WaitGroup
	for idx := range 40 {
		wg.Go(func() {
			tool, args, action := intent("invoice_post", fmt.Sprintf("inv_%d", idx%20))
			key, err := outbox.Admit(tool, args)
			if err != nil {
				return
			}
			_, _ = outbox.Hold(key, action)
		})
	}
	wg.Wait()

	assert.Len(t, outbox.Settle(completed()), 20)
}

type splittingTool struct {
	agentports.AgentTool
}

func (splittingTool) SplitRecords(params map[string]any) ([]map[string]any, error) {
	ids, _ := params["invoiceIds"].([]any)
	calls := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		calls = append(calls, map[string]any{"invoiceIds": []any{id}})
	}

	return calls, nil
}

func TestExpand_SplitsABatchToolsCallIntoOnePerRecord(t *testing.T) {
	t.Parallel()

	calls, err := codemode.Expand(splittingTool{}, map[string]any{
		"invoiceIds": []any{"inv_1", "inv_2", "inv_3"},
	})
	require.NoError(t, err)
	assert.Len(t, calls, 3)

	args := map[string]any{"shipmentId": "shp_1"}
	single, err := codemode.Expand(nil, args)
	require.NoError(t, err)
	assert.Equal(t, []map[string]any{args}, single)
}

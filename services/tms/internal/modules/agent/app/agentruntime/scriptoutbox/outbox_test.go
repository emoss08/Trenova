package scriptoutbox_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/scriptoutbox"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptStep = "script-step"

func intent(tool string, id string) (scriptoutbox.Call, *agentports.PendingAction) {
	args := map[string]any{"invoiceIds": []any{id}}
	call := scriptoutbox.Call{
		Tool:      tool,
		Arguments: args,
		Target: &agentports.ToolTarget{
			Resource: permission.ResourceInvoice,
			ID:       pulid.ID("inv_" + id),
		},
	}

	return call, &agentports.PendingAction{
		ToolName:  tool,
		Arguments: args,
		Tier:      agent.TierActWithApproval,
		Egress:    agent.EgressMoney,
		HeldBy:    []string{agenttoolpolicy.HeldByScriptIntent},
	}
}

func hold(t *testing.T, outbox *scriptoutbox.Outbox, tool string, id string) int {
	t.Helper()

	call, action := intent(tool, id)
	admission, err := outbox.Admit(call)
	require.NoError(t, err)
	seq, err := outbox.Hold(admission, action)
	require.NoError(t, err)

	return seq
}

func completed() scriptoutbox.Settlement {
	return scriptoutbox.Settlement{Completed: true}
}

func settled(
	t *testing.T,
	outbox *scriptoutbox.Outbox,
	settlement scriptoutbox.Settlement,
) []agentports.PendingAction {
	t.Helper()

	filed, err := outbox.Settle(settlement)
	require.NoError(t, err)

	return filed
}

func TestOutbox_FilesEveryIntentOnceInTheOrderPrepared(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	for idx := range 7 {
		assert.Equal(t, idx+1, hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx)))
	}

	filed := settled(t, outbox, completed())
	require.Len(t, filed, 7)
	for idx, action := range filed {
		assert.Equal(t, []any{fmt.Sprintf("inv_%d", idx)}, action.Arguments["invoiceIds"])
		assert.Equal(t, scriptStep, action.ScriptStepKey)
		assert.False(t, action.Executed)
	}
	_, err := outbox.Settle(completed())
	require.ErrorIs(t, err, scriptoutbox.ErrOutboxFiled, "a batch is filed once")
}

func TestOutbox_FilesNothingForAScriptThatDidNotFinish(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")
	hold(t, outbox, "invoice_post", "inv_2")

	assert.Empty(t, settled(t, outbox, scriptoutbox.Settlement{Completed: false}))
	call, _ := intent("invoice_post", "inv_3")
	_, err := outbox.Admit(call)
	require.ErrorIs(t, err, scriptoutbox.ErrOutboxFiled)
}

func TestOutbox_FoldsAnIdenticalIntentIntoTheFirst(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	for range 2 {
		call, action := intent("invoice_post", "inv_1")
		admission, err := outbox.Admit(call)
		require.NoError(t, err)
		assert.Equal(t, 1, admission.Folded, "the same change asked for again is change 1")
		seq, err := outbox.Hold(admission, action)
		require.NoError(t, err)
		assert.Equal(t, 1, seq)
	}
	assert.Equal(t, 2, hold(t, outbox, "invoice_send", "inv_1"),
		"another tool on the same record is another change")

	assert.Len(t, settled(t, outbox, completed()), 2)
}

func TestOutbox_RefusesTheBatchForTwoDifferentChangesToOneRecord(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")
	hold(t, outbox, "invoice_post", "inv_2")

	call, _ := intent("invoice_post", "inv_1")
	call.Arguments = map[string]any{"invoiceIds": []any{"inv_1"}, "note": "post it twice"}
	_, err := outbox.Admit(call)
	require.ErrorIs(t, err, scriptoutbox.ErrConflictingIntents)
	assert.Contains(t, err.Error(), "changes 1 and 3", "both changes are named")

	call, _ = intent("invoice_post", "inv_9")
	_, err = outbox.Admit(call)
	require.ErrorIs(t, err, scriptoutbox.ErrConflictingIntents, "a refused batch takes nothing more")

	filed, err := outbox.Settle(completed())
	require.ErrorIs(t, err, scriptoutbox.ErrConflictingIntents)
	assert.Empty(t, filed, "neither change is picked: nothing is filed")
}

func TestOutbox_RefusesAnIntentPastTheBatchsSize(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 0)
	for idx := range scriptoutbox.MaxBatchSteps {
		hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx))
	}

	call, _ := intent("invoice_post", "inv_over")
	_, err := outbox.Admit(call)
	require.ErrorIs(t, err, scriptoutbox.ErrBatchTooLarge)
	assert.Len(t, settled(t, outbox, completed()), scriptoutbox.MaxBatchSteps)
}

func TestOutbox_HoldsTheAgentsChangeLimitWhenItIsSmaller(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 3)
	for idx := range 3 {
		hold(t, outbox, "invoice_post", fmt.Sprintf("inv_%d", idx))
	}

	call, _ := intent("invoice_post", "inv_over")
	_, err := outbox.Admit(call)
	require.ErrorIs(t, err, scriptoutbox.ErrBatchTooLarge)
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

			outbox := scriptoutbox.NewOutbox(scriptStep, 500)
			call, action := intent("invoice_post", "inv_1")
			mutate(action)
			admission, err := outbox.Admit(call)
			require.NoError(t, err)
			_, err = outbox.Hold(admission, action)
			require.ErrorIs(t, err, scriptoutbox.ErrIntentRan)
			assert.Zero(t, outbox.Len())
		})
	}
}

func TestOutbox_HoldsACopyTheCallerCannotChange(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	call, action := intent("invoice_post", "inv_1")
	admission, err := outbox.Admit(call)
	require.NoError(t, err)
	_, err = outbox.Hold(admission, action)
	require.NoError(t, err)
	call.Arguments["invoiceIds"] = []any{"inv_other"}
	action.Tier = agent.TierAutoExecute

	filed := settled(t, outbox, completed())
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

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")
	hold(t, outbox, "invoice_post", "inv_2")

	redecided := 0
	filed := settled(t, outbox, scriptoutbox.Settlement{
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

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	filed := settled(t, outbox, scriptoutbox.Settlement{Completed: true, ExternalContent: true})
	require.Len(t, filed, 1)
	assert.Equal(t, agent.TierPropose, filed[0].Tier)
	assert.True(t, filed[0].Tainted)
	assert.Contains(t, filed[0].HeldBy, agenttoolpolicy.HeldByTainted)
}

func TestOutbox_NeverFilesAnIntentThatRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	hold(t, outbox, "invoice_post", "inv_1")

	filed := settled(t, outbox, scriptoutbox.Settlement{
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

	outbox := scriptoutbox.NewOutbox(scriptStep, 500)
	var wg sync.WaitGroup
	for idx := range 40 {
		wg.Go(func() {
			call, action := intent("invoice_post", fmt.Sprintf("inv_%d", idx%20))
			admission, err := outbox.Admit(call)
			if err != nil {
				return
			}
			_, _ = outbox.Hold(admission, action)
		})
	}
	wg.Wait()

	assert.Len(t, settled(t, outbox, completed()), 20)
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

	calls, err := scriptoutbox.Expand(splittingTool{}, map[string]any{
		"invoiceIds": []any{"inv_1", "inv_2", "inv_3"},
	})
	require.NoError(t, err)
	assert.Len(t, calls, 3)

	args := map[string]any{"shipmentId": "shp_1"}
	single, err := scriptoutbox.Expand(nil, args)
	require.NoError(t, err)
	assert.Equal(t, []map[string]any{args}, single)
}

package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scheduleAnswers(workerID string) map[string]*agentports.GraphQLReadResult {
	return map[string]*agentports.GraphQLReadResult{
		"WorkerShiftAssignments": {Data: map[string]any{"workerShiftAssignments": []any{
			map[string]any{
				"id": pulid.MustNew("wsa_").String(), "workerId": workerID,
				"effectiveFrom": 1_800_000_000,
				"shiftTemplate": map[string]any{"name": "Early days"},
			},
		}}},
		opWorkerAvailability: {Data: map[string]any{"workerAvailabilityPreferences": []any{
			map[string]any{"dayOfWeek": 6, "preference": "Unavailable"},
		}}},
		opShiftSwapRequests: {Data: map[string]any{"shiftSwapRequests": []any{
			map[string]any{"id": pulid.MustNew("sswp_").String(), "status": "Accepted"},
		}}},
	}
}

func TestWorkerGetSchedule_ReadsTheWorkersAssignmentsPreferencesAndOpenSwaps(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	reader := &keyedReader{answers: scheduleAnswers(workerID)}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolWorkerGetSchedule), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{paramWorkerID: workerID}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{paramWorkerID: workerID},
		reader.sent("WorkerShiftAssignments").Variables)
	assert.Equal(t, map[string]any{paramWorkerID: workerID},
		reader.sent(opWorkerAvailability).Variables)
	assert.Equal(t, map[string]any{
		paramWorkerID: workerID, wfParamOpenOnly: true, paramLimit: openSwapsShown,
	}, reader.sent(opShiftSwapRequests).Variables, "only the open swaps, as the tool read them")
	outcome, _ := data.(map[string]any)
	items, _ := outcome[outcomeItems].([]map[string]any)
	require.Len(t, items, 1)
	assert.Len(t, outcome["preferences"], 1)
	assert.Len(t, outcome["openSwaps"], 1)
	assert.Equal(t, permission.ResourceWorkerSchedule, tool.Policy().Resource)
	assert.Equal(t, []string{paramWorkerID}, tool.ParamSchema()["required"])
}

// A reader who may not read swaps gets the schedule without them, and the
// swaps are never read, as the repository tool read them only for one.
func TestWorkerGetSchedule_ReadsNoSwapsForAReaderWhoMayNotReadThem(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_").String()
	reader := &keyedReader{answers: scheduleAnswers(workerID)}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolWorkerGetSchedule), reader,
		&mocks.DenyResourcesPermissionEngine{Denied: []string{permission.ResourceShiftSwap.String()}})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{paramWorkerID: workerID}))
	require.NoError(t, err)

	assert.Nil(t, reader.sent(opShiftSwapRequests))
	outcome, _ := data.(map[string]any)
	assert.NotContains(t, outcome, "openSwaps")
	assert.Len(t, outcome["preferences"], 1)
}

// The resolver's team scope governs: a schedule the person may not read
// is a refusal, and nothing is read beside it.
func TestWorkerGetSchedule_AReadTheServerRefusesIsARefusal(t *testing.T) {
	t.Parallel()

	answers := scheduleAnswers(pulid.MustNew("wrk_").String())
	answers["WorkerShiftAssignments"] = &agentports.GraphQLReadResult{
		Errors: []agentports.GraphQLReadError{{
			Code: agentports.GraphQLCodeForbidden, Message: "outside your team",
		}},
	}
	reader := &keyedReader{answers: answers}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolWorkerGetSchedule), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	_, err = tool.Query(t.Context(),
		keyedCall(map[string]any{paramWorkerID: pulid.MustNew("wrk_").String()}))
	var refused *agentports.ReadRefusedError
	require.ErrorAs(t, err, &refused)
	assert.Nil(t, reader.sent(opWorkerAvailability))
	assert.Nil(t, reader.sent(opShiftSwapRequests))
}

func TestListShiftTemplates_ReadsTheActivePatternsAndSpellsEachOut(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"ShiftTemplates": {Data: map[string]any{"shiftTemplates": []any{map[string]any{
			"id": pulid.MustNew("shft_").String(), "code": "EARLY", "name": "Early days",
			"daysOfWeek": "0111110", "startMinute": float64(360),
			"durationMinutes": float64(630), "cycleWeeks": float64(1),
		}}}},
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListShiftTemplates), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{paramActiveOnly: true, paramLimit: graphQLListLimit},
		reader.sent("ShiftTemplates").Variables, "the active patterns, a page of them")
	outcome, _ := data.(map[string]any)
	items, _ := outcome[outcomeItems].([]map[string]any)
	require.Len(t, items, 1)
	assert.Equal(t, "Mon Tue Wed Thu Fri", items[0]["days"])
	assert.Equal(t, "06:00", items[0]["starts"])
	assert.Equal(t, "10:30", items[0]["hours"])

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{paramActiveOnly: false}))
	require.NoError(t, err)
	assert.Equal(t, false, reader.requests[1].Variables[paramActiveOnly],
		"a call may ask for the retired patterns too")
}

func TestListDOTRandomDraws_ReadsTheRoundsWithTheActivePoolsBeside(t *testing.T) {
	t.Parallel()

	poolID := pulid.MustNew("drpool_").String()
	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DotRandomDraws": {Data: map[string]any{"dotRandomDraws": []any{map[string]any{
			"id": pulid.MustNew("drdraw_").String(), "poolId": poolID, "status": "Final",
		}}}},
		opDOTRandomPools: {Data: map[string]any{"dotRandomPools": map[string]any{
			"edges":    []any{map[string]any{"node": map[string]any{"id": poolID}}},
			"pageInfo": map[string]any{"hasNextPage": false},
		}}},
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, toolListDOTRandomDraws), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)

	assert.Empty(t, reader.sent("DotRandomDraws").Variables, "every pool's rounds")
	assert.Equal(t, map[string]any{"input": map[string]any{
		inputStatus: activeStatus, inputFirst: alongsideMost,
	}}, reader.sent(opDOTRandomPools).Variables)
	outcome, _ := data.(map[string]any)
	assert.Len(t, outcome["pools"], 1)
	assert.Len(t, outcome[outcomeItems], 1)

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{paramPoolID: poolID}))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{paramPoolID: poolID}, reader.requests[2].Variables)
}

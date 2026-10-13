package movetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const checkCallNow = int64(1_800_000_000)

type fakeCheckCalls struct {
	move   *shipment.ShipmentMove
	guard  agenttoolstest.WriteGuard
	logged []*checkcallports.LogCheckCallRequest
	refuse error
}

func (f *fakeCheckCalls) plan(
	req *checkcallports.LogCheckCallRequest,
) *checkcallports.CheckCallPreview {
	reportedAt := checkCallNow
	if req.Details.ReportedAt != nil {
		reportedAt = *req.Details.ReportedAt
	}
	return &checkcallports.CheckCallPreview{
		Move: f.move,
		CheckCall: &checkcall.CheckCall{
			ShipmentMoveID:   req.MoveID,
			Source:           req.Source,
			SourceKey:        req.SourceKey,
			Kind:             req.Details.Kind,
			StopID:           req.Details.StopID,
			ReportedAt:       reportedAt,
			City:             req.Details.City,
			StateCode:        req.Details.StateCode,
			Delayed:          req.Details.Delayed,
			DelayReason:      req.Details.DelayReason,
			EstimatedArrival: req.Details.EstimatedArrival,
			Note:             req.Details.Note,
			ContactName:      req.Details.ContactName,
		},
	}
}

func (f *fakeCheckCalls) PreviewLog(
	_ context.Context,
	req *checkcallports.LogCheckCallRequest,
) (*checkcallports.CheckCallPreview, error) {
	if f.refuse != nil {
		return nil, f.refuse
	}
	return f.plan(req), nil
}

func (f *fakeCheckCalls) Log(
	_ context.Context,
	req *checkcallports.LogCheckCallRequest,
) (*checkcallports.CheckCallResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	if f.refuse != nil {
		return nil, f.refuse
	}
	f.logged = append(f.logged, req)
	return &checkcallports.CheckCallResult{CheckCall: f.plan(req).CheckCall}, nil
}

func TestLogCheckCall_AnUpdateIsLoggedAsTheAgentUnderTheCallKey(t *testing.T) {
	t.Parallel()

	calls := &fakeCheckCalls{}
	tool := newLogCheckCallTool(calls, nil, nil)
	moveID := pulid.MustNew("sm_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":      moveID.String(),
		"kind":        "Update",
		"stopId":      pulid.MustNew("stp_").String(),
		"city":        "Tulsa",
		"stateCode":   "OK",
		"delayed":     true,
		"delayReason": "Weather on I-44",
		"contactName": "Jo",
	})
	params.IdempotencyKey = "run_1:call_4"

	require.NoError(t, tool.Execute(t.Context(), params))

	require.Len(t, calls.logged, 1)
	logged := calls.logged[0]
	assert.Equal(t, moveID, logged.MoveID)
	assert.Equal(t, checkcall.SourceAgent, logged.Source)
	assert.Equal(t, "run_1:call_4", logged.SourceKey)
	assert.Equal(t, params.Actor.UserID, logged.ReportedByID)
	assert.Equal(t, params.OrganizationID, logged.TenantInfo.OrgID)
	assert.Equal(t, checkcall.KindUpdate, logged.Details.Kind)
	assert.True(t, logged.Details.StopID.IsNil(), "an update is about the load, not a stop")
	assert.Equal(t, "Tulsa", logged.Details.City)
	assert.True(t, logged.Details.Delayed)
	assert.Equal(t, "Weather on I-44", logged.Details.DelayReason)
	assert.Nil(t, logged.Details.ReportedAt, "no time given means now")
}

func TestLogCheckCall_AStopVisitNeedsTheStop(t *testing.T) {
	t.Parallel()

	calls := &fakeCheckCalls{}
	tool := newLogCheckCallTool(calls, nil, nil)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("sm_").String(),
		"kind":   "Arrival",
	}))

	require.Error(t, err)
	assert.Empty(t, calls.logged)

	err = tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("sm_").String(),
		"kind":   "Arrived",
	}))
	require.Error(t, err)
}

func TestLogCheckCall_ReadsTimesWhereTheNextStopIs(t *testing.T) {
	t.Parallel()

	moveStops := &fakeMoveStops{move: &shipment.ShipmentMove{Stops: []*shipment.Stop{
		{
			ID:       pulid.MustNew("stp_"),
			Sequence: 2,
			Location: &location.Location{Timezone: "America/Chicago"},
		},
		{
			ID:              pulid.MustNew("stp_"),
			Sequence:        1,
			ActualDeparture: new(int64),
			Location:        &location.Location{Timezone: "America/New_York"},
		},
	}}}
	calls := &fakeCheckCalls{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":           pulid.MustNew("sm_").String(),
		"kind":             "Update",
		"reportedAt":       "2026-09-30T06:15",
		"estimatedArrival": "2026-09-30T11:00",
	})
	params.Timezone = "America/Los_Angeles"

	require.NoError(t, newLogCheckCallTool(calls, nil, moveStops).Execute(t.Context(), params))

	logged := calls.logged[0].Details
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Chicago", 2026, 9, 30, 6, 15),
		*logged.ReportedAt, "the next open stop's zone wins over the organization's")
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Chicago", 2026, 9, 30, 11, 0),
		*logged.EstimatedArrival)
}

func TestLogCheckCall_PreviewOfAnUpdateMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	calls := &fakeCheckCalls{move: move}
	tool := newLogCheckCallTool(calls, &savingMoves{move: move}, nil).(*LogCheckCallTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":      move.ID.String(),
		"kind":        "Update",
		"city":        "Elko",
		"stateCode":   "NV",
		"delayed":     true,
		"delayReason": "Chain law on I-80",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&calls.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	require.Len(t, preview.Changes, 1)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentMove, change.Resource)
	assert.Equal(t, "Check call", change.Label)
	assert.Equal(t, "Elko, NV", agenttoolstest.FieldByPath(t, change, "location").After)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "reportedAt").Volatile)
	assert.Contains(t, preview.Summary, "running late")

	request, err := tool.request(t.Context(), &params)
	require.NoError(t, err)
	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change,
		checkCallViewOf(calls.plan(calls.logged[0])), checkCallOptions(request)...)
}

func TestLogCheckCall_PreviewOfAnArrivalShowsTheStopItSettles(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	calls := &fakeCheckCalls{move: move}
	moves := &savingMoves{move: move}
	tool := newLogCheckCallTool(calls, moves, nil).(*LogCheckCallTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"moveId":     move.ID.String(),
				"kind":       "Arrival",
				"stopId":     move.Stops[0].ID.String(),
				"reportedAt": "2026-09-21T14:13:20",
			}))
		},
	)

	require.Len(t, preview.Changes, 2)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Pickup at Reno Cold Storage",
		agenttoolstest.FieldByPath(t, created, "stop").After)
	assert.False(t, agenttoolstest.FieldByPath(t, created, "reportedAt").Volatile)
	stop := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, move.ID, stop.EntityID)
	assert.Equal(t, "InTransit", agenttoolstest.FieldByPath(t, stop, "moveStatus").After)
	assert.Contains(t, preview.Summary, "from Assigned to InTransit")
	assert.Empty(t, calls.logged)
}

func TestLogCheckCall_PreviewWarnsWhenTheServiceWouldRefuse(t *testing.T) {
	t.Parallel()

	move := dispatchedMove()
	calls := &fakeCheckCalls{
		move:   move,
		refuse: errortypes.NewBusinessError("This load is no longer being tracked"),
	}
	tool := newLogCheckCallTool(calls, &savingMoves{move: move}, nil).(*LogCheckCallTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&calls.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"moveId": move.ID.String(),
				"kind":   "Update",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestLogCheckCall_NeedsApprovalAndTargetsTheMove(t *testing.T) {
	t.Parallel()

	tool := newLogCheckCallTool(&fakeCheckCalls{}, nil, nil)
	policy := tool.Policy()
	assert.Equal(t, agent.TierActWithApproval, policy.DefaultTier)
	assert.Equal(t, permission.ResourceShipmentMove, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)

	moveID := pulid.MustNew("sm_")
	target, ok := tool.(*LogCheckCallTool).Target(map[string]any{"moveId": moveID.String()})
	require.True(t, ok)
	assert.Equal(t, moveID, target.ID)
}

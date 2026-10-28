package movetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMoveService struct {
	shipmentmoveports.ShipmentMoveService

	recorded *shipmentmoveports.RecordStopActualRequest
	err      error
}

func (f *fakeMoveService) RecordStopActual(
	_ context.Context,
	req *shipmentmoveports.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	f.recorded = req

	return &shipment.ShipmentMove{ID: req.MoveID}, f.err
}

type movesReporter struct {
	moves shipmentmoveports.ShipmentMoveService
	key   string
	via   trackingevent.Source
}

func (r *movesReporter) RecordReported(
	ctx context.Context,
	params *trackingeventports.RecordReportedStopEventParams,
) (*trackingeventports.StopEventResult, error) {
	r.key = params.SourceKey
	r.via = params.Source
	move, err := r.moves.RecordStopActual(ctx, &shipmentmoveports.RecordStopActualRequest{
		TenantInfo: params.TenantInfo,
		MoveID:     params.MoveID,
		StopID:     params.StopID,
		Action:     params.Action,
		OccurredAt: params.OccurredAt,
		DepartedAt: params.DepartedAt,
	})
	if err != nil {
		return nil, err
	}
	return &trackingeventports.StopEventResult{Move: move}, nil
}

func reportsFor(moves shipmentmoveports.ShipmentMoveService) stopReporter {
	if moves == nil {
		return nil
	}
	return &movesReporter{moves: moves}
}

/*
Recording an arrival or a departure is what advances a shipment: the move's
status, the stop's actuals and everything downstream of them follow from it.
That makes it the most useful dispatch write and the one most worth getting
exactly right — a departure recorded against the wrong stop moves a load that
has not moved.
*/
func TestRecordStopActual_PassesTheActionThrough(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"Arrive", "Depart"} {
		moves := &fakeMoveService{}
		tool := newRecordStopActualTool(moves, reportsFor(moves), nil)

		moveID := pulid.MustNew("smv_")
		stopID := pulid.MustNew("stp_")

		require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
			"moveId": moveID.String(),
			"stopId": stopID.String(),
			"action": action,
		})))

		require.NotNil(t, moves.recorded)
		assert.Equal(t, moveID, moves.recorded.MoveID)
		assert.Equal(t, stopID, moves.recorded.StopID)
		assert.Equal(t, shipmentmoveports.StopActualAction(action), moves.recorded.Action)
	}
}

// Anything that is not one of the two actions is refused rather than guessed
// at. "Arrived" is not "Arrive", and silently coercing it would record the
// wrong event.
func TestRecordStopActual_RefusesAnActionItDoesNotKnow(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrived",
	}))
	require.Error(t, err)
	assert.Nil(t, moves.recorded)
}

/*
Leaving occurredAt unset means the service stamps it now, which is right when a
dispatcher is reporting something as it happens. A model inventing a timestamp
for an event it was told about after the fact would put a precise-looking lie on
the record, so the field is only sent when the caller supplied one.
*/
func TestRecordStopActual_OnlySendsATimeWhenGivenOne(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})))
	assert.Nil(t, moves.recorded.OccurredAt, "no time given means the service stamps it")

	moves2 := &fakeMoveService{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":     pulid.MustNew("smv_").String(),
		"stopId":     pulid.MustNew("stp_").String(),
		"action":     "Depart",
		"occurredAt": "2026-09-30T14:30",
	})
	params.Timezone = "America/Chicago"
	require.NoError(
		t,
		newRecordStopActualTool(moves2, reportsFor(moves2), nil).Execute(t.Context(), params),
	)
	require.NotNil(t, moves2.recorded.OccurredAt)
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Chicago", 2026, 9, 30, 14, 30),
		*moves2.recorded.OccurredAt)
}

type fakeMoveStops struct {
	move *shipment.ShipmentMove
	asks int
}

func (f *fakeMoveStops) GetByID(
	_ context.Context,
	req *shipmentmoveports.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	f.asks++
	f.move.ID = req.MoveID

	return f.move, nil
}

func TestRecordStopActual_ReadsATimeWhereTheStopIs(t *testing.T) {
	t.Parallel()

	stopID := pulid.MustNew("stp_")
	moveStops := &fakeMoveStops{move: &shipment.ShipmentMove{Stops: []*shipment.Stop{{
		ID:       stopID,
		Location: &location.Location{Timezone: "America/Los_Angeles"},
	}}}}
	moves := &fakeMoveService{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId":     pulid.MustNew("smv_").String(),
		"stopId":     stopID.String(),
		"action":     "Arrive",
		"occurredAt": "2026-09-30T06:15",
	})
	params.Timezone = "America/New_York"

	require.NoError(
		t,
		newRecordStopActualTool(moves, reportsFor(moves), moveStops).Execute(t.Context(), params),
	)

	require.NotNil(t, moves.recorded.OccurredAt)
	assert.Equal(t, agenttoolstest.LocalInstant(t, "America/Los_Angeles", 2026, 9, 30, 6, 15),
		*moves.recorded.OccurredAt, "the stop's own zone wins over the organization's")
}

func TestRecordStopActual_RefusesAUnixTimeOrAnOffset(t *testing.T) {
	t.Parallel()

	for _, occurredAt := range []any{float64(1789000000), "1789000000", "2026-09-30T14:30:00Z"} {
		moves := &fakeMoveService{}
		err := newRecordStopActualTool(
			moves,
			reportsFor(moves),
			nil,
		).Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
			"moveId":     pulid.MustNew("smv_").String(),
			"stopId":     pulid.MustNew("stp_").String(),
			"action":     "Arrive",
			"occurredAt": occurredAt,
		}))

		require.Error(t, err, occurredAt)
		var fieldErr *errortypes.Error
		require.ErrorAs(t, err, &fieldErr)
		assert.Equal(t, "occurredAt", fieldErr.Field)
		assert.Nil(t, moves.recorded, "nothing is recorded at a guessed time")
	}
}

func TestRecordStopActual_ScopesToTheActorTenant(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})
	require.NoError(t, tool.Execute(t.Context(), params))

	assert.Equal(t, params.OrganizationID, moves.recorded.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, moves.recorded.TenantInfo.BuID)
}

func TestRecordStopActual_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves, reportsFor(moves), nil)

	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrTenantMismatch)
	assert.Nil(t, moves.recorded)
}

// An actual is a claim about the physical world. It is not reversible by
// re-recording, and it needs a person's eyes before it becomes fact.
func TestRecordStopActual_IsIrreversibleAndNeedsApproval(t *testing.T) {
	t.Parallel()

	tool := newRecordStopActualTool(nil, reportsFor(nil), nil)

	assert.False(t, tool.Policy().Reversible)
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().DefaultTier)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)
}

func TestRecordStopActual_ReportsAsTheAgentUnderTheCallKey(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	reports := &movesReporter{moves: moves}
	tool := newRecordStopActualTool(moves, reports, nil)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})
	params.IdempotencyKey = "run_1:call_7"

	require.NoError(t, tool.Execute(t.Context(), params))

	assert.Equal(t, trackingevent.SourceAgent, reports.via)
	assert.Equal(t, "run_1:call_7", reports.key)
	require.NotNil(t, moves.recorded)
}

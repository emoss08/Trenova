package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMoveService struct {
	serviceports.ShipmentMoveService

	recorded *repositories.RecordStopActualRequest
	err      error
}

func (f *fakeMoveService) RecordStopActual(
	_ context.Context,
	req *repositories.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	f.recorded = req

	return &shipment.ShipmentMove{ID: req.MoveID}, f.err
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
		tool := newRecordStopActualTool(moves, nil)

		moveID := pulid.MustNew("smv_")
		stopID := pulid.MustNew("stp_")

		require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
			"moveId": moveID.String(),
			"stopId": stopID.String(),
			"action": action,
		})))

		require.NotNil(t, moves.recorded)
		assert.Equal(t, moveID, moves.recorded.MoveID)
		assert.Equal(t, stopID, moves.recorded.StopID)
		assert.Equal(t, repositories.StopActualAction(action), moves.recorded.Action)
	}
}

// Anything that is not one of the two actions is refused rather than guessed
// at. "Arrived" is not "Arrive", and silently coercing it would record the
// wrong event.
func TestRecordStopActual_RefusesAnActionItDoesNotKnow(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves, nil)

	err := tool.Execute(t.Context(), executeParams(map[string]any{
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
	tool := newRecordStopActualTool(moves, nil)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})))
	assert.Nil(t, moves.recorded.OccurredAt, "no time given means the service stamps it")

	moves2 := &fakeMoveService{}
	params := executeParams(map[string]any{
		"moveId":     pulid.MustNew("smv_").String(),
		"stopId":     pulid.MustNew("stp_").String(),
		"action":     "Depart",
		"occurredAt": "2026-09-30T14:30",
	})
	params.Timezone = "America/Chicago"
	require.NoError(t, newRecordStopActualTool(moves2, nil).Execute(t.Context(), params))
	require.NotNil(t, moves2.recorded.OccurredAt)
	assert.Equal(t, localInstant(t, "America/Chicago", 2026, 9, 30, 14, 30),
		*moves2.recorded.OccurredAt)
}

type fakeMoveStops struct {
	move *shipment.ShipmentMove
	asks int
}

func (f *fakeMoveStops) GetByID(
	_ context.Context,
	req *repositories.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	f.asks++
	f.move.ID = req.MoveID

	return f.move, nil
}

func localInstant(t *testing.T, zone string, year, month, day, hour, minute int) int64 {
	t.Helper()

	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)

	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc).Unix()
}

func TestRecordStopActual_ReadsATimeWhereTheStopIs(t *testing.T) {
	t.Parallel()

	stopID := pulid.MustNew("stp_")
	moveStops := &fakeMoveStops{move: &shipment.ShipmentMove{Stops: []*shipment.Stop{{
		ID:       stopID,
		Location: &location.Location{Timezone: "America/Los_Angeles"},
	}}}}
	moves := &fakeMoveService{}
	params := executeParams(map[string]any{
		"moveId":     pulid.MustNew("smv_").String(),
		"stopId":     stopID.String(),
		"action":     "Arrive",
		"occurredAt": "2026-09-30T06:15",
	})
	params.Timezone = "America/New_York"

	require.NoError(t, newRecordStopActualTool(moves, moveStops).Execute(t.Context(), params))

	require.NotNil(t, moves.recorded.OccurredAt)
	assert.Equal(t, localInstant(t, "America/Los_Angeles", 2026, 9, 30, 6, 15),
		*moves.recorded.OccurredAt, "the stop's own zone wins over the organization's")
}

func TestRecordStopActual_RefusesAUnixTimeOrAnOffset(t *testing.T) {
	t.Parallel()

	for _, occurredAt := range []any{float64(1789000000), "1789000000", "2026-09-30T14:30:00Z"} {
		moves := &fakeMoveService{}
		err := newRecordStopActualTool(moves, nil).Execute(t.Context(), executeParams(map[string]any{
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
	tool := newRecordStopActualTool(moves, nil)

	params := executeParams(map[string]any{
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
	tool := newRecordStopActualTool(moves, nil)

	params := executeParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrTenantMismatch)
	assert.Nil(t, moves.recorded)
}

// An actual is a claim about the physical world. It is not reversible by
// re-recording, and it needs a person's eyes before it becomes fact.
func TestRecordStopActual_IsIrreversibleAndNeedsApproval(t *testing.T) {
	t.Parallel()

	tool := newRecordStopActualTool(nil, nil)

	assert.False(t, tool.Policy().Reversible)
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().DefaultTier)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)
}

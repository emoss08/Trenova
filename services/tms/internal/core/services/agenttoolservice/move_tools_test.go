package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
		tool := newRecordStopActualTool(moves)

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
	tool := newRecordStopActualTool(moves)

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
	tool := newRecordStopActualTool(moves)

	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"moveId": pulid.MustNew("smv_").String(),
		"stopId": pulid.MustNew("stp_").String(),
		"action": "Arrive",
	})))
	assert.Nil(t, moves.recorded.OccurredAt, "no time given means the service stamps it")

	moves2 := &fakeMoveService{}
	require.NoError(t, newRecordStopActualTool(moves2).Execute(
		t.Context(),
		executeParams(map[string]any{
			"moveId":     pulid.MustNew("smv_").String(),
			"stopId":     pulid.MustNew("stp_").String(),
			"action":     "Depart",
			"occurredAt": float64(1789000000),
		}),
	))
	require.NotNil(t, moves2.recorded.OccurredAt)
	assert.Equal(t, int64(1789000000), *moves2.recorded.OccurredAt)
}

func TestRecordStopActual_ScopesToTheActorTenant(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveService{}
	tool := newRecordStopActualTool(moves)

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
	tool := newRecordStopActualTool(moves)

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

	tool := newRecordStopActualTool(nil)

	assert.False(t, tool.Reversible())
	assert.Equal(t, agent.TierActWithApproval, tool.DefaultAutonomyTier())
	assert.Equal(t, permission.ResourceShipmentMove, tool.PermissionResource())
	assert.Equal(t, permission.OpUpdate, tool.PermissionOperation())
}

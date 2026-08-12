package shipmentmoveservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stopActualMoveForTest() (*shipment.ShipmentMove, *shipment.Stop, *shipment.Stop) {
	origin := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypePickup,
		Status:   shipment.StopStatusNew,
		Sequence: 0,
	}
	destination := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 1,
	}
	move := &shipment.ShipmentMove{
		ID:     pulid.MustNew("sm_"),
		Status: shipment.MoveStatusAssigned,
		Stops:  []*shipment.Stop{origin, destination},
	}
	return move, origin, destination
}

func TestApplyStopActual_NilOccurredAtStampsNow(t *testing.T) {
	t.Parallel()

	move, origin, _ := stopActualMoveForTest()
	before := timeutils.NowUnix()
	stop, err := applyStopActual(move, &repositories.RecordStopActualRequest{
		MoveID: move.ID,
		StopID: origin.ID,
		Action: repositories.StopActualActionArrive,
	})
	after := timeutils.NowUnix()

	require.NoError(t, err)
	require.NotNil(t, stop.ActualArrival)
	assert.GreaterOrEqual(t, *stop.ActualArrival, before)
	assert.LessOrEqual(t, *stop.ActualArrival, after)
	assert.Equal(t, shipment.StopStatusInTransit, stop.Status)
}

func TestApplyStopActual_BackdatedArriveStampsOccurredAt(t *testing.T) {
	t.Parallel()

	move, origin, _ := stopActualMoveForTest()
	occurredAt := timeutils.NowUnix() - 3600
	stop, err := applyStopActual(move, &repositories.RecordStopActualRequest{
		MoveID:     move.ID,
		StopID:     origin.ID,
		Action:     repositories.StopActualActionArrive,
		OccurredAt: &occurredAt,
	})

	require.NoError(t, err)
	require.NotNil(t, stop.ActualArrival)
	assert.Equal(t, occurredAt, *stop.ActualArrival)
}

func TestApplyStopActual_BackdatedDepartStampsOccurredAt(t *testing.T) {
	t.Parallel()

	move, origin, _ := stopActualMoveForTest()
	arrivedAt := timeutils.NowUnix() - 7200
	origin.ActualArrival = &arrivedAt
	origin.Status = shipment.StopStatusInTransit
	occurredAt := arrivedAt + 1800
	stop, err := applyStopActual(move, &repositories.RecordStopActualRequest{
		MoveID:     move.ID,
		StopID:     origin.ID,
		Action:     repositories.StopActualActionDepart,
		OccurredAt: &occurredAt,
	})

	require.NoError(t, err)
	require.NotNil(t, stop.ActualDeparture)
	assert.Equal(t, occurredAt, *stop.ActualDeparture)
	assert.Equal(t, shipment.StopStatusCompleted, stop.Status)
}

func TestApplyStopActual_ArriveBeforePriorDepartureRejected(t *testing.T) {
	t.Parallel()

	move, origin, destination := stopActualMoveForTest()
	arrivedAt := timeutils.NowUnix() - 7200
	departedAt := timeutils.NowUnix() - 3600
	origin.ActualArrival = &arrivedAt
	origin.ActualDeparture = &departedAt
	origin.Status = shipment.StopStatusCompleted
	occurredAt := departedAt - 600

	stop, err := applyStopActual(move, &repositories.RecordStopActualRequest{
		MoveID:     move.ID,
		StopID:     destination.ID,
		Action:     repositories.StopActualActionArrive,
		OccurredAt: &occurredAt,
	})

	require.Nil(t, stop)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Event time predates the previous stop's departure", err.Error())
}

func TestApplyStopActual_DepartBeforeArrivalRejected(t *testing.T) {
	t.Parallel()

	move, origin, _ := stopActualMoveForTest()
	arrivedAt := timeutils.NowUnix() - 3600
	origin.ActualArrival = &arrivedAt
	origin.Status = shipment.StopStatusInTransit
	occurredAt := arrivedAt - 600

	stop, err := applyStopActual(move, &repositories.RecordStopActualRequest{
		MoveID:     move.ID,
		StopID:     origin.ID,
		Action:     repositories.StopActualActionDepart,
		OccurredAt: &occurredAt,
	})

	require.Nil(t, stop)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Event time predates the arrival at this stop", err.Error())
}

func TestRecordStopActualRequest_ValidateOccurredAt(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	base := func() *repositories.RecordStopActualRequest {
		return &repositories.RecordStopActualRequest{
			TenantInfo: tenantInfo,
			MoveID:     pulid.MustNew("sm_"),
			StopID:     pulid.MustNew("stp_"),
			Action:     repositories.StopActualActionArrive,
		}
	}

	valid := base()
	past := timeutils.NowUnix() - 3600
	valid.OccurredAt = &past
	require.Nil(t, valid.Validate())

	future := base()
	tooLate := timeutils.NowUnix() + repositories.StopActualClockSkewSeconds + 3600
	future.OccurredAt = &tooLate
	multiErr := future.Validate()
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "Occurred at cannot be in the future")

	zeroed := base()
	zero := int64(0)
	zeroed.OccurredAt = &zero
	multiErr = zeroed.Validate()
	require.NotNil(t, multiErr)
	assert.Contains(t, multiErr.Error(), "Occurred at must be a valid timestamp")

	withinSkew := base()
	nearNow := timeutils.NowUnix() + repositories.StopActualClockSkewSeconds - 60
	withinSkew.OccurredAt = &nearNow
	require.Nil(t, withinSkew.Validate())
}

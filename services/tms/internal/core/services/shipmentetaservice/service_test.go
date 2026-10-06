package shipmentetaservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubShipments struct{ entities []*shipment.Shipment }

func (s *stubShipments) ListTrackingShipments(
	context.Context,
	*repositories.ListTrackingShipmentsRequest,
) ([]*shipment.Shipment, error) {
	return s.entities, nil
}

type stubBoard struct {
	moves []*repositories.BoardMove
	calls int
}

func (s *stubBoard) ListBoardMoves(
	_ context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardMove, error) {
	s.calls++
	return s.moves, nil
}

type stubTelematics struct {
	positions []*telematics.VehiclePosition
	tractors  []pulid.ID
}

func (s *stubTelematics) ListVehiclePositions(
	_ context.Context,
	req *repositories.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	s.tractors = req.TractorIDs
	return s.positions, nil
}

func (s *stubTelematics) ListWorkerHOSStates(
	context.Context,
	*repositories.ListWorkerHOSStatesRequest,
) ([]*telematics.WorkerHOSState, error) {
	return nil, nil
}

type stubFailures struct {
	failures []*servicefailure.ServiceFailure
}

func (s *stubFailures) ListUnresolvedByShipmentIDs(
	context.Context,
	*repositories.ServiceFailuresByShipmentIDsRequest,
) ([]*servicefailure.ServiceFailure, error) {
	return s.failures, nil
}

func ptr[T any](v T) *T { return &v }

func trackingFixture(now int64, windowEnd int64) (*shipment.Shipment, pulid.ID, pulid.ID) {
	shipmentID := pulid.MustNew("shp_")
	moveID := pulid.MustNew("smv_")
	tractorID := pulid.MustNew("tr_")

	return &shipment.Shipment{
		ID:     shipmentID,
		Status: shipment.StatusInTransit,
		Moves: []*shipment.ShipmentMove{{
			ID:           moveID,
			Status:       shipment.MoveStatusInTransit,
			CoverageType: shipment.MoveCoverageTypeDriver,
			Stops: []*shipment.Stop{
				{
					ID:                   pulid.MustNew("stp_"),
					Type:                 shipment.StopTypePickup,
					Status:               shipment.StopStatusCompleted,
					Sequence:             1,
					ScheduledWindowStart: now - 7200,
					ActualArrival:        ptr(now - 7000),
					ActualDeparture:      ptr(now - 6000),
				},
				{
					ID:                   pulid.MustNew("stp_"),
					Type:                 shipment.StopTypeDelivery,
					Status:               shipment.StopStatusNew,
					Sequence:             2,
					ScheduledWindowStart: windowEnd - 3600,
					ScheduledWindowEnd:   ptr(windowEnd),
					Location: &location.Location{
						Latitude:  ptr(41.0),
						Longitude: ptr(-87.0),
					},
				},
			},
		}},
	}, moveID, tractorID
}

func TestEtasByShipmentIDsBatchesAndEstimates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	entity, moveID, tractorID := trackingFixture(now.Unix(), now.Unix()+600)

	board := &stubBoard{moves: []*repositories.BoardMove{{
		MoveID:            moveID,
		AssignedTractorID: tractorID,
	}}}
	telem := &stubTelematics{positions: []*telematics.VehiclePosition{{
		TractorID:  tractorID,
		Latitude:   42.0,
		Longitude:  -87.0,
		RecordedAt: now.Unix() - 60,
	}}}
	svc := NewWithDependencies(&Dependencies{
		Shipments:  &stubShipments{entities: []*shipment.Shipment{entity}},
		Board:      board,
		Telematics: telem,
		ServiceFailures: &stubFailures{failures: []*servicefailure.ServiceFailure{{
			ShipmentID: entity.ID,
			ReasonCode: &servicefailure.ReasonCode{Label: "Shipper delay"},
		}}},
		Now: func() time.Time { return now },
	})

	etas, err := svc.EtasByShipmentIDs(t.Context(), pagination.TenantInfo{}, []pulid.ID{entity.ID})
	require.NoError(t, err)

	eta := etas[entity.ID]
	require.NotNil(t, eta)
	assert.Equal(t, shipmenttracking.VerdictLate, eta.Verdict)
	require.NotNil(t, eta.EstimatedArrival)
	require.NotNil(t, eta.SlackMinutes)
	assert.Negative(t, *eta.SlackMinutes)
	assert.Equal(t, "Shipper delay", eta.Reason)
	assert.Equal(t, 1, board.calls)
	assert.Equal(t, []pulid.ID{tractorID}, telem.tractors)
}

func TestEtaOf(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC).Unix()

	entity, _, _ := trackingFixture(now, now-1800)
	snapshot := shipmenttracking.Build(
		shipmenttracking.Input{Shipment: entity, Now: now, Timezone: "UTC"},
	)
	eta := EtaOf(entity, snapshot, "")
	require.NotNil(t, eta)
	assert.Equal(t, shipmenttracking.VerdictLate, eta.Verdict)
	assert.Equal(t, int64(-30), *eta.SlackMinutes)
	assert.NotEmpty(t, eta.Reason)

	onTime, _, _ := trackingFixture(now, now+86400)
	snapshot = shipmenttracking.Build(
		shipmenttracking.Input{Shipment: onTime, Now: now, Timezone: "UTC"},
	)
	eta = EtaOf(onTime, snapshot, "Weather")
	require.NotNil(t, eta)
	assert.Equal(t, shipmenttracking.VerdictUnknown, eta.Verdict)
	assert.Empty(t, eta.Reason)

	onTime.Status = shipment.StatusDelayed
	eta = EtaOf(onTime, snapshot, "Weather")
	assert.Equal(t, shipmenttracking.VerdictLate, eta.Verdict)
	assert.Equal(t, "Weather", eta.Reason)

	canceled, _, _ := trackingFixture(now, now)
	canceled.Status = shipment.StatusCanceled
	assert.Nil(t, EtaOf(canceled, snapshot, ""))
}

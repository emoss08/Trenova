package agentquerytoolservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrOf[T any](v T) *T { return &v }

type fakeTrackingShipments struct {
	shipmentports.ShipmentRepository

	byID    map[pulid.ID]*shipment.Shipment
	listed  []*shipment.Shipment
	lastGet *shipmentports.GetShipmentByIDRequest
}

func (f *fakeTrackingShipments) GetByID(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.lastGet = req
	if sp, ok := f.byID[req.ID]; ok {
		return sp, nil
	}

	return nil, errortypes.NewNotFoundError("Shipment not found")
}

func (f *fakeTrackingShipments) List(
	_ context.Context,
	_ *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.listed}, nil
}

type fakeBoardMoves struct {
	moves  []*shipmentports.BoardMove
	filter *shipmentports.DispatchBoardFilter
}

func (f *fakeBoardMoves) ListBoardMoves(
	_ context.Context,
	filter *shipmentports.DispatchBoardFilter,
) ([]*shipmentports.BoardMove, error) {
	f.filter = filter

	return f.moves, nil
}

type fakeTelematics struct {
	positions []*telematics.VehiclePosition
	hos       []*telematics.WorkerHOSState
	maxAge    int64
	tractors  []pulid.ID
}

func (f *fakeTelematics) ListVehiclePositions(
	_ context.Context,
	req *telematicsports.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	f.maxAge = req.MaxAgeSeconds
	f.tractors = req.TractorIDs
	if len(req.TractorIDs) == 0 {
		return f.positions, nil
	}

	return slices.DeleteFunc(slices.Clone(f.positions), func(p *telematics.VehiclePosition) bool {
		return !slices.Contains(req.TractorIDs, p.TractorID)
	}), nil
}

func (f *fakeTelematics) ListWorkerHOSStates(
	_ context.Context,
	_ pagination.TenantInfo,
	_ []pulid.ID,
	_ int,
) ([]*telematics.WorkerHOSState, error) {
	return f.hos, nil
}

func trackedShipment() (*shipment.Shipment, *shipment.ShipmentMove) {
	move := &shipment.ShipmentMove{
		ID:           pulid.MustNew("smv_"),
		Status:       shipment.MoveStatusInTransit,
		CoverageType: shipment.MoveCoverageTypeDriver,
		Stops: []*shipment.Stop{
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             0,
				Type:                 shipment.StopTypePickup,
				Status:               shipment.StopStatusCompleted,
				ScheduledWindowStart: 1_790_000_000 - 7200,
				ActualArrival:        ptrOf(int64(1_790_000_000 - 7000)),
				Location:             &location.Location{Name: "Dallas Yard", City: "Dallas"},
			},
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             1,
				Type:                 shipment.StopTypeDelivery,
				Status:               shipment.StopStatusNew,
				ScheduledWindowStart: 1_790_000_000 + 3*3600,
				Location: &location.Location{
					Name: "Houston DC", City: "Houston",
					Latitude: ptrOf(29.7604), Longitude: ptrOf(-95.3698),
				},
			},
		},
	}

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S12345",
		Status:    shipment.StatusInTransit,
		Customer:  &customer.Customer{Name: "Acme"},
		Moves:     []*shipment.ShipmentMove{move},
	}, move
}

/*
"Where is S12345" used to take four calls and a guess: the shipment, its
moves, the board for the driver, and a position nobody could reach at all.
One call now reads the stops, the assignment, the tractor's position and the
driver's hours, and says where the truck is and when it should arrive.
*/
type fakeStopEtaReader struct {
	rows      []*stopeta.StopETA
	requested []pulid.ID
}

func (f *fakeStopEtaReader) ListByShipmentIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]*stopeta.StopETA, error) {
	f.requested = append(f.requested, shipmentIDs...)
	return f.rows, nil
}

func TestGetShipmentTracking_ComposesStopsAssignmentPositionAndHours(t *testing.T) {
	t.Parallel()

	sp, move := trackedShipment()
	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")
	shipments := &fakeTrackingShipments{byID: map[pulid.ID]*shipment.Shipment{sp.ID: sp}}
	board := &fakeBoardMoves{moves: []*shipmentports.BoardMove{{
		MoveID:              move.ID,
		AssignedWorkerID:    workerID,
		AssignedWorkerName:  "Maria Ortiz",
		AssignedTractorID:   tractorID,
		AssignedTractorCode: "T-104",
		CoverageType:        "driver",
	}}}
	feed := &fakeTelematics{
		positions: []*telematics.VehiclePosition{
			{
				TractorID:  tractorID,
				Latitude:   30.7235,
				Longitude:  -95.5508,
				RecordedAt: 1_790_000_000 - 120,
			},
			{
				TractorID:  pulid.MustNew("trc_"),
				Latitude:   0,
				Longitude:  0,
				RecordedAt: 1_790_000_000,
			},
		},
		hos: []*telematics.WorkerHOSState{
			{WorkerID: workerID, DriveRemainingMs: 4 * 3_600_000, RecordedAt: 1_790_000_000 - 60},
		},
	}
	delivery := move.Stops[1]
	arrival := int64(1_790_000_000 + 2*3600)
	slack := int64(60)
	etas := &fakeStopEtaReader{rows: []*stopeta.StopETA{{
		ShipmentID:       sp.ID,
		StopID:           delivery.ID,
		EstimatedArrival: &arrival,
		SlackMinutes:     &slack,
		Verdict:          stopeta.VerdictOnTime,
		Basis:            stopeta.BasisPosition,
		DriveMethod:      stopeta.DriveMethodRouted,
		ComputedAt:       1_790_000_000 - 30,
	}}}
	tool := newGetShipmentTrackingTool(shipments, board, feed, etas)

	params := testParams(map[string]any{"shipmentId": sp.ID.String()})
	params.Timezone = "America/Chicago"
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	snapshot, ok := result.(*shipmenttracking.Snapshot)
	require.True(t, ok)
	assert.Equal(t, "S12345", snapshot.ProNumber)
	require.Len(t, snapshot.Moves, 1)
	assert.Equal(t, "Maria Ortiz", snapshot.Moves[0].Driver)
	assert.Equal(t, "T-104", snapshot.Moves[0].Tractor)
	require.NotNil(t, snapshot.Position)
	assert.Equal(
		t,
		tractorID.String(),
		snapshot.Position.TractorID,
		"only the assigned tractor's position is used",
	)
	assert.Equal(t, []pulid.ID{tractorID}, feed.tractors,
		"the read names the assigned tractors rather than reading the fleet")
	require.NotNil(t, snapshot.Driver)
	assert.EqualValues(t, 240, snapshot.Driver.DriveRemainingMinutes)
	require.NotNil(t, snapshot.Estimate)
	assert.Equal(t, shipmenttracking.VerdictOnTime, snapshot.Estimate.Verdict)
	assert.Equal(t, arrival, snapshot.Estimate.EstimatedArrival)
	assert.Equal(t, []pulid.ID{sp.ID}, etas.requested)

	require.NotNil(t, shipments.lastGet)
	assert.True(t, shipments.lastGet.ExpandShipmentDetails)
	assert.True(t, shipments.lastGet.IncludeCustomer)
	require.NotNil(t, board.filter)
	assert.Equal(t, []pulid.ID{move.ID}, board.filter.MoveIDs)
	assert.True(t, board.filter.IncludeCovered)
	assert.Equal(t, positionLookbackSeconds, feed.maxAge)
}

func TestGetShipmentTracking_FindsAShipmentByExactProNumber(t *testing.T) {
	t.Parallel()

	sp, _ := trackedShipment()
	other, _ := trackedShipment()
	other.ProNumber = "S123456"
	shipments := &fakeTrackingShipments{listed: []*shipment.Shipment{other, sp}}
	tool := newGetShipmentTrackingTool(
		shipments,
		&fakeBoardMoves{},
		&fakeTelematics{},
		&fakeStopEtaReader{},
	)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"proNumber": "s12345"}))
	require.NoError(t, err)
	snapshot, ok := result.(*shipmenttracking.Snapshot)
	require.True(t, ok)
	assert.Equal(t, sp.ID.String(), snapshot.ShipmentID)

	shipments.listed = []*shipment.Shipment{other}
	_, err = tool.Query(t.Context(), testParams(map[string]any{"proNumber": "S12345"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "S123456", "a near miss is named rather than silently used")

	_, err = tool.Query(t.Context(), testParams(map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shipmentId or proNumber")
}

type fakeBoard struct {
	board *dispatchconsoleservice.Board
	req   *dispatchconsoleservice.GetBoardRequest
}

func (f *fakeBoard) GetBoard(
	_ context.Context,
	req *dispatchconsoleservice.GetBoardRequest,
) (*dispatchconsoleservice.Board, error) {
	f.req = req

	return f.board, nil
}

func TestGetDispatchBoard_ReadsMovesDriversAndUrgency(t *testing.T) {
	t.Parallel()

	board := &fakeBoard{board: &dispatchconsoleservice.Board{
		WindowStart: 1_790_000_000,
		WindowEnd:   1_790_000_000 + 86_400,
		Summary: &shipmentports.BoardSummary{
			UncoveredMoves:   1,
			LateMoves:        1,
			AvailableDrivers: 1,
		},
		Moves: []*dispatchconsoleservice.BoardMove{
			{
				BoardMove: &shipmentports.BoardMove{
					MoveID: pulid.MustNew(
						"smv_",
					), ShipmentID: pulid.MustNew("shp_"), ProNumber: "S1",
					CustomerName: "Acme", OriginName: "Dallas Yard", OriginCity: "Dallas", OriginState: "TX",
					OriginWindowStart: 1_790_000_000 - 1800, DestinationCity: "Houston", DestinationState: "TX",
					DestinationWindowStart: 1_790_000_000 + 7200,
				},
				Urgency: dispatchconsoleservice.UrgencyLate, MinutesToPickup: -30,
			},
			{
				BoardMove: &shipmentports.BoardMove{
					MoveID: pulid.MustNew(
						"smv_",
					), ProNumber: "S2", AssignedWorkerName: "Maria Ortiz",
					AssignedWorkerID: pulid.MustNew(
						"wrk_",
					), CoverageType: "driver", OriginWindowStart: 1_790_000_000 + 3600,
				},
				Urgency: dispatchconsoleservice.UrgencyNow, IsCovered: true,
			},
		},
		Drivers: []*dispatchconsoleservice.BoardDriver{{
			BoardDriver: &shipmentports.BoardDriver{
				WorkerID:    pulid.MustNew("wrk_"),
				FirstName:   "Dan",
				LastName:    "Lee",
				TractorCode: "T-7",
			},
			Availability:     dispatchconsoleservice.AvailabilityOpen,
			DriveRemainingMs: 6 * 3_600_000, FormattedLocation: "Waco, TX", PositionAt: 1_790_000_000 - 600,
		}},
	}}
	tool := newGetDispatchBoardTool(board)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"hoursAhead": 12, "uncoveredOnly": true}),
	)
	require.NoError(t, err)

	view, ok := result.(boardView)
	require.True(t, ok)
	require.Len(t, view.Moves, 2)
	assert.Equal(t, "Late", view.Moves[0].Urgency)
	assert.Equal(t, "Dallas Yard in Dallas, TX", view.Moves[0].Origin)
	assert.Equal(t, "unassigned", view.Moves[0].Coverage)
	assert.True(t, view.Moves[1].Covered)
	assert.Equal(t, "Maria Ortiz", view.Moves[1].Driver)
	require.Len(t, view.Drivers, 1)
	assert.Equal(t, "Dan Lee", view.Drivers[0].Name)
	assert.EqualValues(t, 360, view.Drivers[0].DriveRemainingMinutes)
	assert.Equal(t, 1, view.Counts["Late"])
	assert.Contains(t, view.Note, "Late 1, Now 1")
	require.NotNil(t, view.Summary)
	assert.Equal(t, 1, view.Summary.LateMoves)

	require.NotNil(t, board.req)
	assert.False(t, board.req.IncludeCovered)
	assert.EqualValues(t, 12*3600, board.req.WindowEnd-board.req.WindowStart)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)
}

type fakeWeather struct {
	features []*weatheralertports.WeatherAlertFeature
}

func (f *fakeWeather) GetActiveAlerts(
	_ context.Context,
	_ pagination.TenantInfo,
) (*weatheralertports.WeatherAlertFeatureCollection, error) {
	return &weatheralertports.WeatherAlertFeatureCollection{Features: f.features}, nil
}

func TestListWeatherAlerts_KeepsSevereAndWorseByDefault(t *testing.T) {
	t.Parallel()

	expires := int64(1_790_000_000 + 7200)
	weather := &fakeWeather{features: []*weatheralertports.WeatherAlertFeature{
		{
			Properties: weatheralertports.WeatherAlertFeatureProperties{
				ID:            pulid.MustNew("wal_"),
				Event:         "Ice Storm Warning",
				Severity:      "Extreme",
				AreaDesc:      "Dallas County, TX",
				AlertCategory: weatheralert.AlertCategoryWinterWeather,
				Expires:       &expires,
			},
		},
		{
			Properties: weatheralertports.WeatherAlertFeatureProperties{
				ID:       pulid.MustNew("wal_"),
				Event:    "Wind Advisory",
				Severity: "Minor",
				AreaDesc: "Harris County, TX",
			},
		},
	}}
	tool := newListWeatherAlertsTool(weather)

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]weatherAlertRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "Ice Storm Warning", rows[0].Event)
	assert.Equal(t, "winter_weather", rows[0].Category)
	assert.NotEmpty(t, rows[0].Expires)

	result, err = tool.Query(
		t.Context(),
		testParams(map[string]any{"severity": "Minor", "query": "harris"}),
	)
	require.NoError(t, err)
	outcome, ok = result.(searchOutcome)
	require.True(t, ok)
	rows, ok = outcome.Items.([]weatherAlertRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "Wind Advisory", rows[0].Event)
}

func TestMonitoringTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	tools := []agentports.AgentQueryTool{
		newGetShipmentTrackingTool(
			&fakeTrackingShipments{},
			&fakeBoardMoves{},
			&fakeTelematics{},
			&fakeStopEtaReader{},
		),
		newGetDispatchBoardTool(&fakeBoard{}),
		newListWeatherAlertsTool(&fakeWeather{}),
	}
	for _, tool := range tools {
		params := testParams(map[string]any{"workerId": pulid.MustNew("wrk_").String()})
		params.Actor.OrganizationID = pulid.MustNew("org_")
		_, err := tool.Query(t.Context(), params)
		require.ErrorIs(t, err, ErrTenantMismatch, tool.Name())
	}
}

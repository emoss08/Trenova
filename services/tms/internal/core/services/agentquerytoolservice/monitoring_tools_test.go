package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrOf[T any](v T) *T { return &v }

type fakeTrackingShipments struct {
	repositories.ShipmentRepository

	byID    map[pulid.ID]*shipment.Shipment
	listed  []*shipment.Shipment
	lastGet *repositories.GetShipmentByIDRequest
}

func (f *fakeTrackingShipments) GetByID(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.lastGet = req
	if sp, ok := f.byID[req.ID]; ok {
		return sp, nil
	}

	return nil, errortypes.NewNotFoundError("Shipment not found")
}

func (f *fakeTrackingShipments) List(
	_ context.Context,
	_ *repositories.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.listed}, nil
}

type fakeBoardMoves struct {
	moves  []*repositories.BoardMove
	filter *repositories.DispatchBoardFilter
}

func (f *fakeBoardMoves) ListBoardMoves(
	_ context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardMove, error) {
	f.filter = filter

	return f.moves, nil
}

type fakeTelematics struct {
	positions []*telematics.VehiclePosition
	hos       []*telematics.WorkerHOSState
	maxAge    int64
}

func (f *fakeTelematics) ListVehiclePositions(
	_ context.Context,
	_ pagination.TenantInfo,
	maxAgeSeconds int64,
) ([]*telematics.VehiclePosition, error) {
	f.maxAge = maxAgeSeconds

	return f.positions, nil
}

func (f *fakeTelematics) ListWorkerHOSStates(
	_ context.Context,
	_ pagination.TenantInfo,
	_ []pulid.ID,
	_ int,
) ([]*telematics.WorkerHOSState, error) {
	return f.hos, nil
}

func (f *fakeTelematics) GetWorkerHOSState(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*telematics.WorkerHOSState, error) {
	for _, state := range f.hos {
		if state.WorkerID == workerID {
			return state, nil
		}
	}

	return nil, nil
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
func TestGetShipmentTracking_ComposesStopsAssignmentPositionAndHours(t *testing.T) {
	t.Parallel()

	sp, move := trackedShipment()
	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")
	shipments := &fakeTrackingShipments{byID: map[pulid.ID]*shipment.Shipment{sp.ID: sp}}
	board := &fakeBoardMoves{moves: []*repositories.BoardMove{{
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
	tool := newGetShipmentTrackingTool(shipments, board, feed)

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
	require.NotNil(t, snapshot.Driver)
	assert.EqualValues(t, 240, snapshot.Driver.DriveRemainingMinutes)
	require.NotNil(t, snapshot.Estimate)
	assert.NotEqual(t, shipmenttracking.VerdictUnknown, snapshot.Estimate.Verdict)

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
	tool := newGetShipmentTrackingTool(shipments, &fakeBoardMoves{}, &fakeTelematics{})

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

func TestListVehiclePositions_NarrowsToTheTractorsAskedFor(t *testing.T) {
	t.Parallel()

	wanted := pulid.MustNew("trc_")
	feed := &fakeTelematics{positions: []*telematics.VehiclePosition{
		{
			TractorID: wanted, Latitude: 32.7, Longitude: -96.8, SpeedMph: 61,
			EngineState: telematics.EngineStateOn, RecordedAt: 1_790_000_000 - 300,
			Tractor: &tractor.Tractor{
				Code:          "T-104",
				PrimaryWorker: &worker.Worker{FirstName: "Maria", LastName: "Ortiz"},
			},
		},
		{TractorID: pulid.MustNew("trc_"), RecordedAt: 1_790_000_000},
	}}
	tool := newListVehiclePositionsTool(feed, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"tractorIds":    []any{wanted.String()},
		"maxAgeMinutes": 30,
	}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]vehiclePositionRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "T-104", rows[0].Tractor)
	assert.Equal(t, "Maria Ortiz", rows[0].Driver)
	assert.EqualValues(t, 30*60, feed.maxAge)
	assert.Contains(t, outcome.SearchedFor, "readings newer than 30 minutes")

	_, err = tool.Query(t.Context(), testParams(map[string]any{"tractorIds": []any{"nope"}}))
	require.Error(t, err)
}

func TestGetWorkerHOS_ReadsTheClocksAndMarksAStaleReading(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	feed := &fakeTelematics{hos: []*telematics.WorkerHOSState{{
		WorkerID:         workerID,
		DutyStatus:       telematics.DutyStatusDriving,
		DriveRemainingMs: 90 * 60_000,
		ShiftRemainingMs: 5 * 3_600_000,
		RecordedAt:       1,
		Worker:           &worker.Worker{FirstName: "Maria", LastName: "Ortiz"},
	}}}
	tool := newGetWorkerHOSTool(feed)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"workerId": workerID.String()}),
	)
	require.NoError(t, err)

	row, ok := result.(workerHOSRow)
	require.True(t, ok)
	assert.Equal(t, "Maria Ortiz", row.Name)
	assert.EqualValues(t, 90, row.DriveRemainingMinutes)
	assert.True(t, row.Stale)
	assert.NotEmpty(t, row.Note)

	_, err = tool.Query(
		t.Context(),
		testParams(map[string]any{"workerId": pulid.MustNew("wrk_").String()}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hours of service")
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
		Summary: &repositories.BoardSummary{
			UncoveredMoves:   1,
			LateMoves:        1,
			AvailableDrivers: 1,
		},
		Moves: []*dispatchconsoleservice.BoardMove{
			{
				BoardMove: &repositories.BoardMove{
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
				BoardMove: &repositories.BoardMove{
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
			BoardDriver: &repositories.BoardDriver{
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
	assert.Equal(t, permission.ResourceShipmentMove, tool.PermissionResource())
}

type fakeFailures struct {
	items   []*servicefailure.ServiceFailure
	request *repositories.ListServiceFailuresRequest
}

func (f *fakeFailures) List(
	_ context.Context,
	req *repositories.ListServiceFailuresRequest,
) (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
	f.request = req

	return &pagination.ListResult[*servicefailure.ServiceFailure]{
		Items: f.items,
		Total: len(f.items),
	}, nil
}

func TestListServiceFailures_FiltersOnStatusAndRendersTheRow(t *testing.T) {
	t.Parallel()

	reasonID := pulid.MustNew("sfrc_")
	failures := &fakeFailures{items: []*servicefailure.ServiceFailure{{
		ID: pulid.MustNew("sf_"), Number: "SF-1", ShipmentID: pulid.MustNew("shp_"),
		ShipmentMoveID: pulid.MustNew("smv_"), StopID: pulid.MustNew("stp_"),
		Type: servicefailure.TypeLateDelivery, Source: servicefailure.SourceDetected,
		Status: servicefailure.StatusOpen, StopType: shipment.StopTypeDelivery,
		ScheduledCutoff: 1_790_000_000, ActualArrival: 1_790_000_000 + 2700, LateMinutes: 45,
		ReasonCodeID: &reasonID, ReasonCode: &servicefailure.ReasonCode{Code: "WX", Label: "Weather"},
		DetectedAt: 1_790_000_000 + 2700, Version: 2,
	}}}
	tool := newListServiceFailuresTool(failures)

	result, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Open"},
	)))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)
	rows := outcome.Items.([]any)
	row, ok := rows[0].(serviceFailureRow)
	require.True(t, ok)
	assert.Equal(t, "SF-1", row.Number)
	assert.EqualValues(t, 45, row.LateMinutes)
	assert.Equal(t, "WX Weather", row.ReasonCode)
	assert.EqualValues(t, 2, row.Version)
	assert.NotEmpty(t, row.ScheduledCutoff)

	require.NotNil(t, failures.request)
	require.Len(t, failures.request.Filter.FieldFilters, 1)
	assert.Equal(t, "status", failures.request.Filter.FieldFilters[0].Field)

	_, err = tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Closed"},
	)))
	require.Error(
		t,
		err,
		"a status outside the set is refused, not passed through to match nothing",
	)
	assert.Equal(t, permission.ResourceServiceFailure, tool.PermissionResource())
}

type fakeReasonCodes struct {
	codes   []*servicefailure.ReasonCode
	request *repositories.ServiceFailureReasonCodeSelectOptionsRequest
}

func (f *fakeReasonCodes) SelectOptions(
	_ context.Context,
	req *repositories.ServiceFailureReasonCodeSelectOptionsRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	f.request = req

	return &pagination.ListResult[*servicefailure.ReasonCode]{Items: f.codes}, nil
}

func TestListServiceFailureReasonCodes_ListsActiveCodesForTheStopKind(t *testing.T) {
	t.Parallel()

	codes := &fakeReasonCodes{codes: []*servicefailure.ReasonCode{
		{
			ID:        pulid.MustNew("sfrc_"),
			Code:      "WX",
			Label:     "Weather",
			Category:  servicefailure.ReasonCategoryWeather,
			AppliesTo: servicefailure.ReasonCodeAppliesToAll,
			Active:    true,
		},
		{ID: pulid.MustNew("sfrc_"), Code: "OLD", Label: "Retired", Active: false},
	}}
	tool := newListServiceFailureReasonCodesTool(codes)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"appliesTo": "Delivery", "query": "wea"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]reasonCodeRow)
	require.True(t, ok)
	require.Len(t, rows, 1, "a retired code is not offered")
	assert.Equal(t, "WX", rows[0].Code)
	assert.Equal(t, "Weather", rows[0].Category)

	require.NotNil(t, codes.request)
	assert.Equal(t, servicefailure.ReasonCodeAppliesToDelivery, codes.request.AppliesTo)
	assert.Equal(t, "wea", codes.request.SelectQueryRequest.Query)
}

type fakeDesk struct {
	entries []*detentionservice.DeskEntry
}

func (f *fakeDesk) ListDesk(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*detentionservice.DeskEntry, error) {
	return f.entries, nil
}

func TestListDetentionDesk_RendersTheOpenOccurrencesByUrgency(t *testing.T) {
	t.Parallel()

	arrived := int64(1_790_000_000 - 5400)
	desk := &fakeDesk{entries: []*detentionservice.DeskEntry{
		{
			Occurrence: &detention.DetentionOccurrence{
				ID: pulid.MustNew(
					"dto_",
				), ShipmentID: pulid.MustNew("shp_"), ShipmentProNumber: "S1",
				CustomerName: "Acme", LocationName: "Houston DC", StopType: shipment.StopTypeDelivery,
				Status: detention.OccurrenceStatusAccruing, NotificationStatus: detention.NotificationStatusPending,
				ArrivedAt: &arrived, FreeTimeExpiresAt: 1_790_000_000 - 1800, BillableMinutes: 30,
				Currency: "USD", RequiresApproval: true,
			},
			MinutesUntilFreeEnds: -30, NoticeWindowOpen: true,
			AmountAtRisk: decimal.NewFromFloat(37.5), Urgency: "NoticeOverdue",
		},
		{
			Occurrence: &detention.DetentionOccurrence{
				ID:     pulid.MustNew("dto_"),
				Status: detention.OccurrenceStatusAccruing,
			},
			Urgency: "NoticeDueSoon",
		},
	}}
	tool := newListDetentionDeskTool(desk)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"urgency": "NoticeOverdue"}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]detentionDeskRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "S1", rows[0].ProNumber)
	assert.Equal(t, "37.50", rows[0].AmountAtRisk)
	assert.True(t, rows[0].NoticeWindowOpen)
	assert.True(t, rows[0].RequiresApproval)
	assert.NotEmpty(t, rows[0].ArrivedAt)
	assert.Equal(t, permission.ResourceDetentionPolicy, tool.PermissionResource())
}

type fakeWeather struct {
	features []*serviceports.WeatherAlertFeature
}

func (f *fakeWeather) GetActiveAlerts(
	_ context.Context,
	_ pagination.TenantInfo,
) (*serviceports.WeatherAlertFeatureCollection, error) {
	return &serviceports.WeatherAlertFeatureCollection{Features: f.features}, nil
}

func TestListWeatherAlerts_KeepsSevereAndWorseByDefault(t *testing.T) {
	t.Parallel()

	expires := int64(1_790_000_000 + 7200)
	weather := &fakeWeather{features: []*serviceports.WeatherAlertFeature{
		{
			Properties: serviceports.WeatherAlertFeatureProperties{
				ID:            pulid.MustNew("wal_"),
				Event:         "Ice Storm Warning",
				Severity:      "Extreme",
				AreaDesc:      "Dallas County, TX",
				AlertCategory: weatheralert.AlertCategoryWinterWeather,
				Expires:       &expires,
			},
		},
		{
			Properties: serviceports.WeatherAlertFeatureProperties{
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

	tools := []serviceports.AgentQueryTool{
		newGetShipmentTrackingTool(&fakeTrackingShipments{}, &fakeBoardMoves{}, &fakeTelematics{}),
		newListVehiclePositionsTool(&fakeTelematics{}, &fakePermissions{}),
		newGetWorkerHOSTool(&fakeTelematics{}),
		newGetDispatchBoardTool(&fakeBoard{}),
		newListServiceFailureReasonCodesTool(&fakeReasonCodes{}),
		newListDetentionDeskTool(&fakeDesk{}),
		newListWeatherAlertsTool(&fakeWeather{}),
	}
	for _, tool := range tools {
		params := testParams(map[string]any{"workerId": pulid.MustNew("wrk_").String()})
		params.Actor.OrganizationID = pulid.MustNew("org_")
		_, err := tool.Query(t.Context(), params)
		require.ErrorIs(t, err, ErrTenantMismatch, tool.Name())
	}
}

// A position is a tractor's, and the tool is gated on tractors; the name of
// the person driving it is a worker's, and a reader who may not open workers
// does not learn who is where from the fleet map either. An agent principal
// reads workers within the platform's allow-list, so it still sees the name.
func TestListVehiclePositions_NamesTheDriverOnlyToAReaderOfWorkers(t *testing.T) {
	t.Parallel()

	feed := &fakeTelematics{positions: []*telematics.VehiclePosition{{
		TractorID: pulid.MustNew("trc_"), RecordedAt: 1_790_000_000,
		Tractor: &tractor.Tractor{
			Code:          "T-104",
			PrimaryWorker: &worker.Worker{FirstName: "Maria", LastName: "Ortiz"},
		},
	}}}
	noWorkers := &fakePermissions{readable: map[string]*serviceports.ResourcePermissionDetail{
		permission.ResourceTractor.String(): {
			Operations: []permission.Operation{permission.OpRead},
		},
	}}

	rows := vehicleRows(
		t,
		newListVehiclePositionsTool(feed, noWorkers),
		testParams(map[string]any{}),
	)
	require.Len(t, rows, 1)
	assert.Equal(t, "T-104", rows[0].Tractor)
	assert.Empty(t, rows[0].Driver, "a reader without worker access is not told who is driving")

	agentParams := testParams(map[string]any{})
	agentParams.Actor = &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agd_"),
		OrganizationID: agentParams.OrganizationID,
		BusinessUnitID: agentParams.BusinessUnitID,
	}
	rows = vehicleRows(t, newListVehiclePositionsTool(feed, noWorkers), agentParams)
	require.Len(t, rows, 1)
	assert.Equal(t, "Maria Ortiz", rows[0].Driver)
}

func vehicleRows(
	t *testing.T,
	tool serviceports.AgentQueryTool,
	params serviceports.QueryToolParams,
) []vehiclePositionRow {
	t.Helper()

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]vehiclePositionRow)
	require.True(t, ok)

	return rows
}

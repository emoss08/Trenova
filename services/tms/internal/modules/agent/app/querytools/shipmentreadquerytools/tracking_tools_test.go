package shipmentreadquerytools

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/dispatchquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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

	byID     map[pulid.ID]*shipment.Shipment
	listed   []*shipment.Shipment
	lastGet  *shipmentports.GetShipmentByIDRequest
	lastList *shipmentports.ListShipmentsRequest
	lists    int
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
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	f.lastList = req
	f.lists++

	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.listed}, nil
}

type fakeBoardMoves struct {
	moves  []*dispatchboardports.BoardMove
	filter *dispatchboardports.DispatchBoardFilter
}

func (f *fakeBoardMoves) ListBoardMoves(
	_ context.Context,
	filter *dispatchboardports.DispatchBoardFilter,
) ([]*dispatchboardports.BoardMove, error) {
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
		Status:       shipmentvocab.MoveStatusInTransit,
		CoverageType: shipmentvocab.MoveCoverageTypeDriver,
		Stops: []*shipment.Stop{
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             0,
				Type:                 shipmentvocab.StopTypePickup,
				Status:               shipmentvocab.StopStatusCompleted,
				ScheduledWindowStart: 1_790_000_000 - 7200,
				ActualArrival:        ptrOf(int64(1_790_000_000 - 7000)),
				Location:             &location.Location{Name: "Dallas Yard", City: "Dallas"},
			},
			{
				ID:                   pulid.MustNew("stp_"),
				Sequence:             1,
				Type:                 shipmentvocab.StopTypeDelivery,
				Status:               shipmentvocab.StopStatusNew,
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
		Status:    shipmentvocab.StatusInTransit,
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
	board := &fakeBoardMoves{moves: []*dispatchboardports.BoardMove{{
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
	tool := NewGetShipmentTrackingTool(shipments, board, feed, etas)

	params := querytooltest.TestParams(map[string]any{"shipmentIds": []any{sp.ID.String()}})
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
	tool := NewGetShipmentTrackingTool(
		shipments,
		&fakeBoardMoves{},
		&fakeTelematics{},
		&fakeStopEtaReader{},
	)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"proNumber": "s12345"}),
	)
	require.NoError(t, err)
	snapshot, ok := result.(*shipmenttracking.Snapshot)
	require.True(t, ok)
	assert.Equal(t, sp.ID.String(), snapshot.ShipmentID)

	shipments.listed = []*shipment.Shipment{other}
	_, err = tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"proNumber": "S12345"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "S123456", "a near miss is named rather than silently used")

	_, err = tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shipmentIds or proNumber")
}

/*
The load monitor read the board, then tracked each late load in its own call:
nine calls before it judged a single service failure, and its twelve-call budget
spent. Every load it names now comes back in one call, from one read of each
source for all of them.
*/
func TestGetShipmentTracking_TracksEveryLoadNamedInOneReadOfEachSource(t *testing.T) {
	t.Parallel()

	first, firstMove := trackedShipment()
	second, secondMove := trackedShipment()
	second.ProNumber = "S67890"
	missing := pulid.MustNew("shp_")
	shipments := &fakeTrackingShipments{listed: []*shipment.Shipment{second, first}}
	board := &fakeBoardMoves{}
	etas := &fakeStopEtaReader{}
	tool := NewGetShipmentTrackingTool(shipments, board, &fakeTelematics{}, etas)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{first.ID.String(), second.ID.String(), missing.String()},
	}))
	require.NoError(t, err)

	table, ok := result.(*trackingOutcome)
	require.True(t, ok)
	require.Len(t, table.Items, 2)
	assert.Equal(t, "S12345", table.Items[0].ProNumber, "rows come in the order named")
	assert.Equal(t, "S67890", table.Items[1].ProNumber)
	assert.NotEmpty(t, table.Items[0].Summary)
	require.NotNil(t, table.Items[0].NextStop)
	assert.Contains(t, table.Note, missing.String())

	assert.Equal(t, 1, shipments.lists, "one read for every load")
	require.NotNil(t, shipments.lastList)
	assert.ElementsMatch(t, []pulid.ID{first.ID, second.ID, missing},
		shipments.lastList.ShipmentOptions.IDs)
	assert.True(t, shipments.lastList.ShipmentOptions.ExpandShipmentDetails)
	require.NotNil(t, board.filter)
	assert.ElementsMatch(t, []pulid.ID{secondMove.ID, firstMove.ID}, board.filter.MoveIDs)
	assert.ElementsMatch(t, []pulid.ID{second.ID, first.ID}, etas.requested)
}

func TestGetShipmentTracking_RefusesMoreLoadsThanOneCallReads(t *testing.T) {
	t.Parallel()

	tool := NewGetShipmentTrackingTool(&fakeTrackingShipments{}, &fakeBoardMoves{},
		&fakeTelematics{}, &fakeStopEtaReader{})
	ids := make([]any, 0, maxShipmentReads+1)
	for range maxShipmentReads + 1 {
		ids = append(ids, pulid.MustNew("shp_").String())
	}

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{"shipmentIds": ids}))
	require.ErrorContains(t, err, "more than 25 ids")

	_, err = tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentIds": []any{pulid.MustNew("shp_").String()}, "proNumber": "S1",
	}))
	require.ErrorContains(t, err, "not both")
}

func TestMonitoringTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	tools := []agenttoolports.AgentQueryTool{
		NewGetShipmentTrackingTool(
			&fakeTrackingShipments{},
			&fakeBoardMoves{},
			&fakeTelematics{},
			&fakeStopEtaReader{},
		),
		dispatchquerytools.NewGetDispatchBoardTool(&querytooltest.FakeBoard{}),
		dispatchquerytools.NewListWeatherAlertsTool(&querytooltest.FakeWeather{}),
	}
	for _, tool := range tools {
		params := querytooltest.TestParams(
			map[string]any{"workerId": pulid.MustNew("wrk_").String()},
		)
		params.Actor.OrganizationID = pulid.MustNew("org_")
		_, err := tool.Query(t.Context(), params)
		require.ErrorIs(t, err, querytool.ErrTenantMismatch, tool.Name())
	}
}

package shipmentetaservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubShipments struct{ entities []*shipment.Shipment }

func (s *stubShipments) ListTrackingShipments(
	context.Context,
	*shipmentports.ListTrackingShipmentsRequest,
) ([]*shipment.Shipment, error) {
	return s.entities, nil
}

type stubBoard struct {
	moves []*shipmentports.BoardMove
}

func (s *stubBoard) ListBoardMoves(
	context.Context,
	*shipmentports.DispatchBoardFilter,
) ([]*shipmentports.BoardMove, error) {
	return s.moves, nil
}

type stubWorkers struct{ workers []*worker.Worker }

func (s *stubWorkers) ListWorkersByIDs(
	context.Context,
	pagination.TenantInfo,
	[]pulid.ID,
) ([]*worker.Worker, error) {
	return s.workers, nil
}

type stubTelematics struct {
	positions []*telematics.VehiclePosition
	states    []*telematics.WorkerHOSState
}

func (s *stubTelematics) ListVehiclePositions(
	context.Context,
	*telematicsports.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	return s.positions, nil
}

func (s *stubTelematics) ListWorkerHOSStates(
	context.Context,
	*telematicsports.ListWorkerHOSStatesRequest,
) ([]*telematics.WorkerHOSState, error) {
	return s.states, nil
}

type stubFailures struct {
	failures []*servicefailure.ServiceFailure
}

func (s *stubFailures) ListUnresolvedByShipmentIDs(
	context.Context,
	*servicefailureports.ServiceFailuresByShipmentIDsRequest,
) ([]*servicefailure.ServiceFailure, error) {
	return s.failures, nil
}

type stubCalls struct {
	latest map[pulid.ID]*checkcall.CheckCall
}

func (s *stubCalls) LatestForMoves(
	context.Context,
	pagination.TenantInfo,
	[]pulid.ID,
) (map[pulid.ID]*checkcall.CheckCall, error) {
	if s.latest == nil {
		return map[pulid.ID]*checkcall.CheckCall{}, nil
	}
	return s.latest, nil
}

type stubControls struct {
	control *dispatchcontrol.DispatchControl
}

func (s *stubControls) GetByOrgID(
	context.Context,
	dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	if s.control == nil {
		return nil, errortypes.NewNotFoundError("Dispatch control not found")
	}
	return s.control, nil
}

type stubRoutes struct{ hoursPerLeg float64 }

func (s *stubRoutes) EstimateLegs(
	_ context.Context,
	_ pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	out := make([]routeestimateports.LegEstimate, len(legs))
	for i := range legs {
		out[i] = routeestimateports.LegEstimate{
			Miles:        s.hoursPerLeg * 50,
			DriveSeconds: int64(s.hoursPerLeg * 3600),
			Routed:       true,
		}
	}
	return out, nil
}

type memoryEtas struct {
	rows     map[pulid.ID]*stopeta.StopETA
	replaced int
}

func newMemoryEtas() *memoryEtas {
	return &memoryEtas{rows: map[pulid.ID]*stopeta.StopETA{}}
}

func (m *memoryEtas) ReplaceForShipments(
	_ context.Context,
	req *stopetaports.ReplaceForShipmentsRequest,
) error {
	m.replaced++
	wanted := make(map[pulid.ID]struct{}, len(req.ShipmentIDs))
	for _, id := range req.ShipmentIDs {
		wanted[id] = struct{}{}
	}
	for stopID, row := range m.rows {
		if _, ok := wanted[row.ShipmentID]; ok {
			delete(m.rows, stopID)
		}
	}
	for _, row := range req.Rows {
		copied := *row
		m.rows[row.StopID] = &copied
	}
	return nil
}

func (m *memoryEtas) ListByShipmentIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]*stopeta.StopETA, error) {
	wanted := make(map[pulid.ID]struct{}, len(shipmentIDs))
	for _, id := range shipmentIDs {
		wanted[id] = struct{}{}
	}
	out := make([]*stopeta.StopETA, 0, len(m.rows))
	for _, row := range m.rows {
		if _, ok := wanted[row.ShipmentID]; ok {
			copied := *row
			out = append(out, &copied)
		}
	}
	sortRows(out)
	return out, nil
}

func sortRows(rows []*stopeta.StopETA) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].StopSequence < rows[j-1].StopSequence; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func (m *memoryEtas) ListAttention(
	context.Context,
	*stopetaports.ListAttentionRequest,
) ([]*stopetaports.AttentionStop, error) {
	return nil, nil
}

func (m *memoryEtas) ListTrackedShipmentIDs(
	context.Context,
	*stopetaports.ListTrackedShipmentIDsRequest,
) ([]pulid.ID, error) {
	return nil, nil
}

func (m *memoryEtas) ListActiveTenants(
	context.Context,
	[]shipment.Status,
) ([]pagination.TenantInfo, error) {
	return nil, nil
}

type recordingObserver struct{ calls [][]pulid.ID }

func (r *recordingObserver) StopEtasRefreshed(
	_ context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) {
	r.calls = append(r.calls, shipmentIDs)
}

func ptr[T any](v T) *T { return &v }

type fixture struct {
	now      time.Time
	tenant   pagination.TenantInfo
	entity   *shipment.Shipment
	delivery *shipment.Stop
	moveID   pulid.ID
	tractor  pulid.ID
	worker   pulid.ID
	etas     *memoryEtas
	observer *recordingObserver
	svc      *Service
	failures *stubFailures
	routes   *stubRoutes
	control  *dispatchcontrol.DispatchControl
}

func newFixture(windowEndFromNow time.Duration) *fixture {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	moveID := pulid.MustNew("smv_")
	tractor := pulid.MustNew("tr_")
	workerID := pulid.MustNew("wrk_")
	windowEnd := now.Add(windowEndFromNow).Unix()
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Type:                 shipment.StopTypeDelivery,
		Status:               shipment.StopStatusNew,
		Sequence:             2,
		ScheduledWindowStart: windowEnd - 3600,
		ScheduledWindowEnd:   ptr(windowEnd),
		Location:             &location.Location{Latitude: ptr(41.0), Longitude: ptr(-87.0)},
	}
	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Status:         shipment.StatusInTransit,
		Customer:       &customer.Customer{},
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
					ScheduledWindowStart: now.Unix() - 7200,
					ActualArrival:        ptr(now.Unix() - 7000),
					ActualDeparture:      ptr(now.Unix() - 6000),
					Location: &location.Location{
						Latitude:  ptr(42.0),
						Longitude: ptr(-88.0),
					},
				},
				delivery,
			},
		}},
	}

	f := &fixture{
		now:      now,
		tenant:   tenant,
		entity:   entity,
		delivery: delivery,
		moveID:   moveID,
		tractor:  tractor,
		worker:   workerID,
		etas:     newMemoryEtas(),
		observer: &recordingObserver{},
		failures: &stubFailures{},
		routes:   &stubRoutes{hoursPerLeg: 2},
		control:  dispatchcontrol.NewDefaultDispatchControl(tenant.OrgID, tenant.BuID),
	}
	f.svc = NewWithDependencies(&Dependencies{
		Shipments: &stubShipments{entities: []*shipment.Shipment{entity}},
		Board: &stubBoard{moves: []*shipmentports.BoardMove{{
			MoveID:            moveID,
			AssignedTractorID: tractor,
			AssignedWorkerID:  workerID,
			CoverageType:      string(shipment.MoveCoverageTypeDriver),
		}}},
		Workers: &stubWorkers{},
		Telematics: &stubTelematics{positions: []*telematics.VehiclePosition{{
			TractorID:  tractor,
			Latitude:   41.5,
			Longitude:  -87.5,
			RecordedAt: now.Unix() - 60,
		}}},
		ServiceFailures: f.failures,
		StopEtas:        f.etas,
		CheckCalls:      &stubCalls{},
		Controls:        &stubControls{control: f.control},
		Routes:          f.routes,
		Observers:       []stopetaports.Observer{f.observer},
		Now:             func() time.Time { return f.now },
	})

	return f
}

func TestRefreshShipments_StoresRowsAndNotifiesOnChange(t *testing.T) {
	t.Parallel()

	f := newFixture(3 * time.Hour)
	result, err := f.svc.RefreshShipments(
		t.Context(),
		f.tenant,
		[]pulid.ID{f.entity.ID, f.entity.ID},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Shipments)
	assert.Equal(t, 1, result.Rows)
	assert.Zero(t, result.Attention)

	row := f.etas.rows[f.delivery.ID]
	require.NotNil(t, row)
	assert.Equal(t, f.now.Unix()+2*3600, *row.EstimatedArrival)
	assert.Equal(t, int64(60), *row.SlackMinutes)
	assert.Equal(t, stopeta.VerdictOnTime, row.Verdict)
	assert.Equal(t, int64(60), row.ThresholdMinutes)
	require.Len(t, f.observer.calls, 1)
	assert.Equal(t, []pulid.ID{f.entity.ID}, f.observer.calls[0])

	_, err = f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)
	assert.Len(t, f.observer.calls, 1, "an unchanged projection is not announced again")
}

func TestRefreshShipments_CustomerThresholdMakesItAtRisk(t *testing.T) {
	t.Parallel()

	f := newFixture(3 * time.Hour)
	f.entity.Customer.EtaRiskThresholdMinutes = ptr(int16(90))

	result, err := f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Attention)
	row := f.etas.rows[f.delivery.ID]
	assert.Equal(t, stopeta.VerdictAtRisk, row.Verdict)
	assert.Equal(t, int64(90), row.ThresholdMinutes)
}

func TestRefreshShipments_FinishedShipmentClearsRows(t *testing.T) {
	t.Parallel()

	f := newFixture(3 * time.Hour)
	_, err := f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)
	require.Len(t, f.etas.rows, 1)

	f.entity.Status = shipment.StatusCompleted
	_, err = f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)
	assert.Empty(t, f.etas.rows)
	assert.Len(t, f.observer.calls, 2)
}

func TestRefreshShipments_RequiresDependencies(t *testing.T) {
	t.Parallel()

	svc := NewWithDependencies(&Dependencies{})
	_, err := svc.RefreshShipments(
		t.Context(),
		pagination.TenantInfo{},
		[]pulid.ID{pulid.MustNew("shp_")},
	)
	require.Error(t, err)
}

func TestEtasByShipmentIDs_ReadsStoredRowsWithFailureReason(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Hour)
	_, err := f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)

	f.failures.failures = []*servicefailure.ServiceFailure{{
		ShipmentID: f.entity.ID,
		ReasonCode: &servicefailure.ReasonCode{Label: "Shipper delay"},
	}}
	etas, err := f.svc.EtasByShipmentIDs(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)

	eta := etas[f.entity.ID]
	require.NotNil(t, eta)
	assert.Equal(t, stopeta.VerdictLate, eta.Verdict)
	assert.Equal(t, f.delivery.ID, eta.StopID)
	require.NotNil(t, eta.SlackMinutes)
	assert.Equal(t, int64(-60), *eta.SlackMinutes)
	assert.Equal(t, "Shipper delay", eta.Reason)
	assert.Len(t, eta.Stops, 1)
}

func TestTrackingSnapshots_UseStoredEstimate(t *testing.T) {
	t.Parallel()

	f := newFixture(3 * time.Hour)
	_, err := f.svc.RefreshShipments(t.Context(), f.tenant, []pulid.ID{f.entity.ID})
	require.NoError(t, err)

	snapshots, err := f.svc.TrackingSnapshots(t.Context(), f.tenant, []pulid.ID{f.entity.ID}, "UTC")
	require.NoError(t, err)
	snapshot := snapshots[f.entity.ID]
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.Estimate)
	assert.Equal(t, stopeta.VerdictOnTime, snapshot.Estimate.Verdict)
	assert.Equal(t, f.now.Unix()+2*3600, snapshot.Estimate.EstimatedArrival)
	assert.Equal(t, "Position", snapshot.Estimate.Source)
}

func TestGoverning(t *testing.T) {
	t.Parallel()

	arrived := &stopeta.StopETA{
		StopID:  pulid.MustNew("stp_"),
		Arrived: true,
		Verdict: stopeta.VerdictLate,
	}
	next := &stopeta.StopETA{StopID: pulid.MustNew("stp_"), Verdict: stopeta.VerdictOnTime}
	risky := &stopeta.StopETA{StopID: pulid.MustNew("stp_"), Verdict: stopeta.VerdictAtRisk}
	late := &stopeta.StopETA{StopID: pulid.MustNew("stp_"), Verdict: stopeta.VerdictLate}

	assert.Same(t, next, stopeta.Governing([]*stopeta.StopETA{arrived, next}))
	assert.Same(t, risky, stopeta.Governing([]*stopeta.StopETA{arrived, next, risky}))
	assert.Same(t, late, stopeta.Governing([]*stopeta.StopETA{next, risky, late}))
	assert.Same(t, arrived, stopeta.Governing([]*stopeta.StopETA{arrived}))
}

func TestRowsDiffer(t *testing.T) {
	t.Parallel()

	stopID := pulid.MustNew("stp_")
	base := &stopeta.StopETA{
		StopID:           stopID,
		Verdict:          stopeta.VerdictOnTime,
		EstimatedArrival: ptr(int64(1_000_020)),
		SlackMinutes:     ptr(int64(30)),
	}
	sameMinute := *base
	sameMinute.EstimatedArrival = ptr(int64(1_000_030))
	assert.False(t, RowsDiffer([]*stopeta.StopETA{base}, []*stopeta.StopETA{&sameMinute}))

	moved := *base
	moved.SlackMinutes = ptr(int64(20))
	assert.True(t, RowsDiffer([]*stopeta.StopETA{base}, []*stopeta.StopETA{&moved}))
	assert.True(t, RowsDiffer(nil, []*stopeta.StopETA{base}))
	assert.False(t, RowsDiffer(nil, nil))
}

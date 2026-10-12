package trackingeventservice_test

import (
	"context"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/trackingeventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/driverportalports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeDB struct {
	kernelports.DBConnection
}

func (fakeDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type memoryRepo struct {
	trackingeventports.TrackingEventRepository
	mu     sync.Mutex
	events []*trackingevent.TrackingEvent
}

func (r *memoryRepo) Insert(
	_ context.Context,
	entity *trackingevent.TrackingEvent,
) (*trackingevent.TrackingEvent, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.events {
		if existing.Source == entity.Source && existing.SourceKey == entity.SourceKey {
			copied := *existing
			return &copied, false, nil
		}
	}
	stored := *entity
	r.events = append(r.events, &stored)
	copied := stored
	return &copied, true, nil
}

func (r *memoryRepo) ListByMove(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) ([]*trackingevent.TrackingEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*trackingevent.TrackingEvent, 0, len(r.events))
	for _, event := range r.events {
		if event.ShipmentMoveID == moveID {
			copied := *event
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *memoryRepo) RecordVerdicts(
	_ context.Context,
	_ pagination.TenantInfo,
	verdicts []trackingeventports.TrackingEventVerdict,
	at int64,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, verdict := range verdicts {
		for _, event := range r.events {
			if event.ID == verdict.ID {
				event.Outcome = verdict.Outcome
				event.OutcomeReason = verdict.Reason
				event.UpdatedAt = at
			}
		}
	}
	return nil
}

func (r *memoryRepo) byKey(key string) *trackingevent.TrackingEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, event := range r.events {
		if event.SourceKey == key {
			return event
		}
	}
	return nil
}

type memoryMoves struct {
	shipmentports.ShipmentMoveService
	move     *shipment.ShipmentMove
	recorded []*shipmentports.RecordStopActualRequest
}

func (m *memoryMoves) ReconcileStopActuals(
	ctx context.Context,
	req *shipmentports.ReconcileStopActualsRequest,
) (*shipmentports.ReconcileStopActualsResult, error) {
	var applied []shipment.StopActualChange
	for range 8 {
		changes, err := req.Plan(ctx, m.move)
		if err != nil {
			return nil, err
		}
		if len(changes) == 0 {
			return &shipmentports.ReconcileStopActualsResult{Move: m.move, Changes: applied}, nil
		}
		if _, err = m.move.ApplyStopActualChanges(changes); err != nil {
			return nil, err
		}
		applied = append(applied, changes...)
	}
	return &shipmentports.ReconcileStopActualsResult{Move: m.move, Changes: applied}, nil
}

func (m *memoryMoves) RecordStopActual(
	_ context.Context,
	req *shipmentports.RecordStopActualRequest,
) (*shipment.ShipmentMove, error) {
	m.recorded = append(m.recorded, req)
	change := shipment.StopActualChange{StopID: req.StopID}
	if req.Action == shipmentports.StopActualActionArrive {
		change.Arrival = req.OccurredAt
		change.Departure = req.DepartedAt
	} else {
		change.Departure = req.OccurredAt
	}
	if _, err := m.move.ApplyStopActualChanges([]shipment.StopActualChange{change}); err != nil {
		return nil, err
	}
	return m.move, nil
}

type harness struct {
	service *trackingeventservice.Service
	repo    *memoryRepo
	moves   *memoryMoves
	tenant  pagination.TenantInfo
	move    *shipment.ShipmentMove
	stops   []*shipment.Stop
	ship    *shipment.Shipment
}

func newHarness(t *testing.T, stopCount int) *harness {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ship := &shipment.Shipment{
		ID:                    pulid.MustNew("shp_"),
		Status:                shipment.StatusInTransit,
		BillingTransferStatus: shipment.BillingTransferNone,
	}
	move := &shipment.ShipmentMove{
		ID:         pulid.MustNew("smv_"),
		ShipmentID: ship.ID,
		Status:     shipment.MoveStatusAssigned,
	}
	stops := make([]*shipment.Stop, 0, stopCount)
	for i := range stopCount {
		stop := &shipment.Stop{
			ID:         pulid.MustNew("stp_"),
			Sequence:   int64(i + 1),
			LocationID: pulid.MustNew("loc_"),
			Status:     shipment.StopStatusNew,
		}
		stops = append(stops, stop)
		move.Stops = append(move.Stops, stop)
	}

	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(move, nil).Maybe()
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(ship, nil).Maybe()

	repo := &memoryRepo{}
	moves := &memoryMoves{move: move}
	service := trackingeventservice.New(trackingeventservice.Params{
		DB:           fakeDB{},
		Logger:       zap.NewNop(),
		Repo:         repo,
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
		MoveService:  moves,
	})

	return &harness{
		service: service,
		repo:    repo,
		moves:   moves,
		tenant:  tenant,
		move:    move,
		stops:   stops,
		ship:    ship,
	}
}

func (h *harness) observed(
	stop int,
	kind shipment.VisitKind,
	source trackingevent.Source,
	key string,
	at int64,
) *trackingeventports.RecordObservedStopEventParams {
	return &trackingeventports.RecordObservedStopEventParams{
		TenantInfo:  h.tenant,
		MoveID:      h.move.ID,
		StopID:      h.stops[stop].ID,
		Kind:        kind,
		Source:      source,
		SourceKey:   key,
		MatchMethod: trackingevent.MatchLocation,
		EventAt:     at,
	}
}

func actuals(stops []*shipment.Stop) [][2]int64 {
	out := make([][2]int64, 0, len(stops))
	for _, stop := range stops {
		var pair [2]int64
		if stop.ActualArrival != nil {
			pair[0] = *stop.ActualArrival
		}
		if stop.ActualDeparture != nil {
			pair[1] = *stop.ActualDeparture
		}
		out = append(out, pair)
	}
	return out
}

func TestRecordObserved_OrderAndRepeatsDoNotChangeTheFinalActuals(t *testing.T) {
	t.Parallel()

	base := timeutils.NowUnix() - 10*3600
	type input struct {
		stop   int
		kind   shipment.VisitKind
		source trackingevent.Source
		key    string
		at     int64
	}
	inputs := []input{
		{0, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:1", base},
		{0, shipment.VisitArrival, trackingevent.SourceEDI, "edi:a:0", base + 120},
		{0, shipment.VisitDeparture, trackingevent.SourceTelematics, "samsara:2", base + 3600},
		{0, shipment.VisitDeparture, trackingevent.SourceEDI, "edi:a:1", base + 3500},
		{1, shipment.VisitArrival, trackingevent.SourceEDI, "edi:b:0", base + 2*3600},
		{1, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:3", base + 2*3600 + 60},
		{1, shipment.VisitDeparture, trackingevent.SourceTelematics, "samsara:4", base + 3*3600},
		{2, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:5", base + 5*3600},
		{2, shipment.VisitDeparture, trackingevent.SourceEDI, "edi:c:0", base + 6*3600},
	}

	run := func(t *testing.T, order []input) [][2]int64 {
		t.Helper()
		h := newHarness(t, 3)
		for _, in := range order {
			_, err := h.service.RecordObserved(
				t.Context(),
				h.observed(in.stop, in.kind, in.source, in.key, in.at),
			)
			require.NoError(t, err)
		}
		return actuals(h.stops)
	}

	want := run(t, inputs)
	assert.Equal(t, [][2]int64{
		{base, base + 3600},
		{base + 2*3600 + 60, base + 3*3600},
		{base + 5*3600, base + 6*3600},
	}, want)

	rng := rand.New(rand.NewPCG(7, 11))
	for i := range 50 {
		order := slices.Clone(inputs)
		rng.Shuffle(len(order), func(a, b int) { order[a], order[b] = order[b], order[a] })
		if i%2 == 0 {
			order = append(order, order[:len(order)/2]...)
		}
		require.Equal(t, want, run(t, order), "order %d", i)
	}
}

func TestRecordObserved_ARepeatIsReplayedWithoutReconciling(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	at := timeutils.NowUnix() - 3600
	params := h.observed(0, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:9", at)

	first, err := h.service.RecordObserved(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeApplied, first.Outcome)
	assert.False(t, first.Replayed)

	again, err := h.service.RecordObserved(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, again.Replayed)
	assert.Equal(t, trackingevent.OutcomeApplied, again.Outcome)
	assert.Len(t, h.repo.events, 1)
}

func TestRecordObserved_AnEventAheadOfAnOpenStopWaits(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	at := timeutils.NowUnix() - 3600

	result, err := h.service.RecordObserved(
		t.Context(),
		h.observed(1, shipment.VisitArrival, trackingevent.SourceEDI, "edi:x:0", at),
	)
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomePending, result.Outcome)
	assert.NotEmpty(t, result.Reason)
	assert.Nil(t, h.stops[1].ActualArrival)

	for idx, kind := range []shipment.VisitKind{shipment.VisitArrival, shipment.VisitDeparture} {
		_, err = h.service.RecordObserved(t.Context(), h.observed(
			0,
			kind,
			trackingevent.SourceTelematics,
			"samsara:"+strconv.Itoa(idx),
			at-3000+int64(idx)*600,
		))
		require.NoError(t, err)
	}

	require.NotNil(t, h.stops[1].ActualArrival)
	assert.Equal(t, at, *h.stops[1].ActualArrival)
	assert.Equal(t, trackingevent.OutcomeApplied, h.repo.byKey("edi:x:0").Outcome)
}

func TestRecordObserved_AnInvoicedLoadKeepsItsActuals(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	h.ship.Status = shipment.StatusInvoiced
	at := timeutils.NowUnix() - 3600

	result, err := h.service.RecordObserved(
		t.Context(),
		h.observed(0, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:1", at),
	)
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeSuperseded, result.Outcome)
	assert.Equal(t, "This load has been invoiced", result.Reason)
	assert.Nil(t, h.stops[0].ActualArrival)
}

func TestRecordObserved_ACanceledLoadRefusesTheEvent(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	h.move.Status = shipment.MoveStatusCanceled
	at := timeutils.NowUnix() - 3600

	result, err := h.service.RecordObserved(
		t.Context(),
		h.observed(0, shipment.VisitArrival, trackingevent.SourceEDI, "edi:z:0", at),
	)
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeRefused, result.Outcome)
	assert.Equal(t, trackingevent.OutcomeRefused, h.repo.byKey("edi:z:0").Outcome)
}

func TestRecordObserved_RejectsInteractiveSourcesAndBadInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	at := timeutils.NowUnix() - 60

	for name, params := range map[string]*trackingeventports.RecordObservedStopEventParams{
		"interactive source": h.observed(
			0, shipment.VisitArrival, trackingevent.SourceDriver, "k", at,
		),
		"future time": h.observed(
			0, shipment.VisitArrival, trackingevent.SourceEDI, "k", at+86400,
		),
		"bad kind": h.observed(0, "Dwell", trackingevent.SourceEDI, "k", at),
	} {
		_, err := h.service.RecordObserved(t.Context(), params)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsRefusal(err), name)
	}

	_, err := h.service.RecordObserved(
		t.Context(),
		&trackingeventports.RecordObservedStopEventParams{
			TenantInfo:  h.tenant,
			MoveID:      h.move.ID,
			StopID:      pulid.MustNew("stp_"),
			Kind:        shipment.VisitArrival,
			Source:      trackingevent.SourceEDI,
			SourceKey:   "k",
			MatchMethod: trackingevent.MatchLocation,
			EventAt:     at,
		},
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, h.repo.events)
}

func TestRecordReported_RecordsArrivalAndDepartureOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	arrived := timeutils.NowUnix() - 7200
	departed := arrived + 1800
	params := &trackingeventports.RecordReportedStopEventParams{
		TenantInfo:   h.tenant,
		MoveID:       h.move.ID,
		StopID:       h.stops[0].ID,
		Action:       shipmentports.StopActualActionArrive,
		OccurredAt:   &arrived,
		DepartedAt:   &departed,
		Source:       trackingevent.SourceAgent,
		SourceKey:    "run_1:step_2",
		ReportedByID: pulid.MustNew("usr_"),
	}

	result, err := h.service.RecordReported(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeApplied, result.Outcome)
	assert.False(t, result.Replayed)
	require.Len(t, h.repo.events, 2)
	assert.Equal(t, shipment.VisitDeparture, h.repo.byKey("run_1:step_2:departure").Kind)
	assert.Equal(t, [2]int64{arrived, departed}, actuals(h.stops)[0])

	again, err := h.service.RecordReported(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, again.Replayed)
	assert.Len(t, h.moves.recorded, 1)
	assert.Len(t, h.repo.events, 2)
}

func TestRecordReported_AKeyReusedForAnotherStopIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	at := timeutils.NowUnix() - 600
	params := &trackingeventports.RecordReportedStopEventParams{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		StopID:     h.stops[0].ID,
		Action:     shipmentports.StopActualActionArrive,
		OccurredAt: &at,
		Source:     trackingevent.SourceDriver,
		SourceKey:  "device:42",
	}
	_, err := h.service.RecordReported(t.Context(), params)
	require.NoError(t, err)

	params.StopID = h.stops[1].ID
	_, err = h.service.RecordReported(t.Context(), params)
	require.Error(t, err)
	assert.True(t, errortypes.IsRefusal(err))
}

func TestRecordReported_ValidatesTheDepartureTime(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	at := timeutils.NowUnix() - 600
	for name, params := range map[string]*trackingeventports.RecordReportedStopEventParams{
		"departure with a depart action": {
			TenantInfo: h.tenant, MoveID: h.move.ID, StopID: h.stops[0].ID,
			Action: shipmentports.StopActualActionDepart, OccurredAt: &at, DepartedAt: &at,
			Source: trackingevent.SourceDispatcher,
		},
		"departure without an arrival time": {
			TenantInfo: h.tenant, MoveID: h.move.ID, StopID: h.stops[0].ID,
			Action: shipmentports.StopActualActionArrive, DepartedAt: &at,
			Source: trackingevent.SourceDispatcher,
		},
		"machine source": {
			TenantInfo: h.tenant, MoveID: h.move.ID, StopID: h.stops[0].ID,
			Action: shipmentports.StopActualActionArrive, Source: trackingevent.SourceEDI,
		},
	} {
		_, err := h.service.RecordReported(t.Context(), params)
		require.Error(t, err, name)
	}
	assert.Empty(t, h.repo.events)
}

func TestRecordStopVisit_MapsTheTrackingOutcome(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	at := timeutils.NowUnix() - 3600
	event := &telematics.TelematicsEvent{
		ID:         telematics.NewEventID(),
		Provider:   "Samsara",
		EventID:    "geo-1",
		EventType:  telematics.EventType("GeofenceEntry"),
		OccurredAt: at,
	}

	verdict, err := h.service.RecordStopVisit(t.Context(), &telematicsports.StopVisitReport{
		TenantInfo:   h.tenant,
		Event:        event,
		MoveID:       h.move.ID,
		StopID:       h.stops[1].ID,
		ProviderStop: true,
	})
	require.NoError(t, err)
	assert.Equal(t, telematics.StopOutcomePending, verdict.Outcome)

	stored := h.repo.byKey("Samsara:geo-1")
	require.NotNil(t, stored)
	assert.Equal(t, trackingevent.SourceTelematics, stored.Source)
	assert.Equal(t, trackingevent.MatchStopReference, stored.MatchMethod)
	assert.Equal(t, "telematics_event:"+event.ID.String(), stored.RawReference)

	event.EventID = "geo-2"
	verdict, err = h.service.RecordStopVisit(t.Context(), &telematicsports.StopVisitReport{
		TenantInfo: h.tenant,
		Event:      event,
		MoveID:     h.move.ID,
		StopID:     h.stops[0].ID,
	})
	require.NoError(t, err)
	assert.Equal(t, telematics.StopOutcomeRecorded, verdict.Outcome)
	assert.Equal(t, trackingevent.MatchLocation, h.repo.byKey("Samsara:geo-2").MatchMethod)

	_, err = h.service.RecordStopVisit(t.Context(), &telematicsports.StopVisitReport{})
	require.Error(t, err)
}

func TestReportDriverStop_RecordsATapOnceUnderTheDriversKey(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 2)
	driver := pulid.MustNew("usr_")
	tapped := timeutils.NowUnix() - 900
	report := &driverportalports.StopReport{
		TenantInfo:    h.tenant,
		MoveID:        h.move.ID,
		StopID:        h.stops[0].ID,
		OccurredAt:    &tapped,
		ClientEventID: "7f1c2a",
		ReportedByID:  driver,
	}

	require.NoError(t, h.service.ReportDriverStop(t.Context(), report))
	require.NoError(t, h.service.ReportDriverStop(t.Context(), report))

	stored := h.repo.byKey(driver.String() + ":7f1c2a")
	require.NotNil(t, stored)
	assert.Equal(t, trackingevent.SourceDriver, stored.Source)
	assert.Equal(t, shipment.VisitArrival, stored.Kind)
	assert.Equal(t, driver, stored.ReportedByID)
	assert.Equal(t, tapped, *h.stops[0].ActualArrival)
	assert.Len(t, h.moves.recorded, 1)
}

func TestReportDriverStop_AFastDeviceClockIsReadAsNow(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	ahead := timeutils.NowUnix() + 3*3600
	before := timeutils.NowUnix()

	require.NoError(t, h.service.ReportDriverStop(t.Context(), &driverportalports.StopReport{
		TenantInfo:   h.tenant,
		MoveID:       h.move.ID,
		StopID:       h.stops[0].ID,
		OccurredAt:   &ahead,
		ReportedByID: pulid.MustNew("usr_"),
	}))

	require.NotNil(t, h.stops[0].ActualArrival)
	assert.GreaterOrEqual(t, *h.stops[0].ActualArrival, before)
	assert.Less(t, *h.stops[0].ActualArrival, ahead)
}

func TestReportDriverStop_RejectsABadClientEventID(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	for _, id := range []string{"has space", strings.Repeat("x", 129)} {
		err := h.service.ReportDriverStop(t.Context(), &driverportalports.StopReport{
			TenantInfo:    h.tenant,
			MoveID:        h.move.ID,
			StopID:        h.stops[0].ID,
			Departure:     true,
			ClientEventID: id,
		})
		require.Error(t, err, id)
		assert.True(t, errortypes.IsRefusal(err), id)
	}
	assert.Empty(t, h.repo.events)
}

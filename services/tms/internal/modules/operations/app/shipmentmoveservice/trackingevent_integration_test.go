//go:build integration

package shipmentmoveservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/trackingeventrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/trackingeventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type trackingHarness struct {
	tracking   *trackingeventservice.Service
	moveRepo   shipmentmoveports.ShipmentMoveRepository
	tenantInfo pagination.TenantInfo
	move       *shipment.ShipmentMove
	shipmentID pulid.ID
}

func newTrackingHarness(t *testing.T, bol string) (context.Context, *trackingHarness) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx, svc, shipmentRepo, moveRepo, tenantInfo, fixture, controlRepo := newIntegrationService(
		t,
		ctx,
		db,
	)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("tenantports.GetShipmentControlRequest")).
		Return(&shipmentcontrol.ShipmentControl{}, nil).
		Maybe()
	graph := testutil.CreateShipmentGraph(
		t,
		ctx,
		db,
		fixture,
		tenantInfo,
		testutil.ShipmentGraphParams{
			BOL:          bol,
			ProNumber:    "PRO-" + bol,
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusAssigned},
		},
	)

	conn := postgres.NewTestConnection(db)
	tracking := trackingeventservice.New(trackingeventservice.Params{
		DB:     conn,
		Logger: zap.NewNop(),
		Repo: trackingeventrepository.New(
			trackingeventrepository.Params{DB: conn, Logger: zap.NewNop()},
		),
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
		MoveService:  svc,
	})

	return ctx, &trackingHarness{
		tracking:   tracking,
		moveRepo:   moveRepo,
		tenantInfo: tenantInfo,
		move:       graph.Moves[0],
		shipmentID: graph.Shipment.ID,
	}
}

type observedReport struct {
	stop   int
	kind   shipment.VisitKind
	source trackingevent.Source
	key    string
	at     int64
}

func (h *trackingHarness) record(
	t *testing.T,
	ctx context.Context,
	report observedReport,
) *trackingeventports.StopEventResult {
	t.Helper()
	result, err := h.tracking.RecordObserved(ctx, &trackingeventports.RecordObservedStopEventParams{
		TenantInfo:  h.tenantInfo,
		MoveID:      h.move.ID,
		StopID:      h.move.Stops[report.stop].ID,
		Kind:        report.kind,
		Source:      report.source,
		SourceKey:   report.key,
		MatchMethod: trackingevent.MatchLocation,
		EventAt:     report.at,
	})
	require.NoError(t, err)
	return result
}

func (h *trackingHarness) persisted(t *testing.T, ctx context.Context) *shipment.ShipmentMove {
	t.Helper()
	move, err := h.moveRepo.GetByID(ctx, &shipmentmoveports.GetMoveByIDRequest{
		MoveID:            h.move.ID,
		TenantInfo:        h.tenantInfo,
		ExpandMoveDetails: true,
	})
	require.NoError(t, err)
	return move
}

func persistedActuals(move *shipment.ShipmentMove) [][2]int64 {
	out := make([][2]int64, 0, len(move.Stops))
	for _, stop := range move.ActiveStopsBySequence() {
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

func TestTrackingEventsIntegrationSettleTheSameInAnyOrder(t *testing.T) {
	t.Parallel()

	base := timeutils.NowUnix() - 12*3600
	reports := []observedReport{
		{0, shipment.VisitArrival, trackingevent.SourceTelematics, "samsara:a1", base},
		{0, shipment.VisitArrival, trackingevent.SourceEDI, "edi:p:x3", base + 300},
		{0, shipment.VisitDeparture, trackingevent.SourceTelematics, "samsara:d1", base + 3600},
		{1, shipment.VisitArrival, trackingevent.SourceEDI, "edi:d:x1", base + 4*3600},
		{1, shipment.VisitDeparture, trackingevent.SourceEDI, "edi:d:d1", base + 5*3600},
	}
	want := [][2]int64{{base, base + 3600}, {base + 4*3600, base + 5*3600}}

	for name, order := range map[string][]int{
		"in order":                 {0, 1, 2, 3, 4},
		"reversed with repeats":    {4, 3, 2, 1, 0, 4, 0},
		"delivery before pickup":   {3, 4, 1, 0, 2},
		"departure before arrival": {2, 4, 0, 3, 1, 2},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, h := newTrackingHarness(t, "BOL-TKEV-"+pulid.MustNew("x_").String()[2:12])

			for _, index := range order {
				h.record(t, ctx, reports[index])
			}

			move := h.persisted(t, ctx)
			assert.Equal(t, want, persistedActuals(move))
			assert.Equal(t, shipment.MoveStatusCompleted, move.Status)

			events, err := h.tracking.ListForShipment(
				ctx,
				&trackingeventports.ListTrackingEventsByShipmentRequest{
					TenantInfo: h.tenantInfo,
					ShipmentID: h.shipmentID,
				},
			)
			require.NoError(t, err)
			require.Len(t, events, len(reports))
			outcomes := make(map[string]trackingevent.Outcome, len(events))
			for _, event := range events {
				outcomes[event.SourceKey] = event.Outcome
			}
			assert.Equal(t, map[string]trackingevent.Outcome{
				"samsara:a1": trackingevent.OutcomeApplied,
				"edi:p:x3":   trackingevent.OutcomeSuperseded,
				"samsara:d1": trackingevent.OutcomeApplied,
				"edi:d:x1":   trackingevent.OutcomeApplied,
				"edi:d:d1":   trackingevent.OutcomeApplied,
			}, outcomes)
		})
	}
}

func TestTrackingEventsIntegrationAWaitingArrivalLandsWithTheEarlierDeparture(t *testing.T) {
	t.Parallel()

	ctx, h := newTrackingHarness(t, "BOL-TKEV-WAIT")
	base := timeutils.NowUnix() - 6*3600

	waiting := h.record(
		t,
		ctx,
		observedReport{
			1,
			shipment.VisitArrival,
			trackingevent.SourceEDI,
			"edi:w:x1",
			base + 3*3600,
		},
	)
	assert.Equal(t, trackingevent.OutcomePending, waiting.Outcome)
	assert.Nil(t, h.persisted(t, ctx).ActiveStopsBySequence()[1].ActualArrival)

	again := h.record(
		t,
		ctx,
		observedReport{
			1,
			shipment.VisitArrival,
			trackingevent.SourceEDI,
			"edi:w:x1",
			base + 3*3600,
		},
	)
	assert.True(t, again.Replayed)

	h.record(
		t,
		ctx,
		observedReport{
			0,
			shipment.VisitArrival,
			trackingevent.SourceTelematics,
			"samsara:w1",
			base,
		},
	)
	h.record(
		t,
		ctx,
		observedReport{
			0,
			shipment.VisitDeparture,
			trackingevent.SourceTelematics,
			"samsara:w2",
			base + 3600,
		},
	)

	move := h.persisted(t, ctx)
	assert.Equal(t, [][2]int64{{base, base + 3600}, {base + 3*3600, 0}}, persistedActuals(move))
	assert.Equal(t, shipment.MoveStatusInTransit, move.Status)
}

func TestTrackingEventsIntegrationADispatcherReportOutranksTelematics(t *testing.T) {
	t.Parallel()

	ctx, h := newTrackingHarness(t, "BOL-TKEV-DISP")
	base := timeutils.NowUnix() - 6*3600

	h.record(
		t,
		ctx,
		observedReport{
			0,
			shipment.VisitArrival,
			trackingevent.SourceTelematics,
			"samsara:q1",
			base + 600,
		},
	)
	arrived := base
	result, err := h.tracking.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
		StopID:     h.move.Stops[0].ID,
		Action:     shipmentmoveports.StopActualActionArrive,
		OccurredAt: &arrived,
		Source:     trackingevent.SourceDispatcher,
		SourceKey:  "idem-1",
	})
	require.Error(
		t,
		err,
		"the stop already has an arrival, so the strict path refuses a second one",
	)
	assert.Nil(t, result)

	move := h.persisted(t, ctx)
	assert.Equal(t, base+600, *move.ActiveStopsBySequence()[0].ActualArrival)

	departed := base + 7200
	result, err = h.tracking.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
		StopID:     h.move.Stops[0].ID,
		Action:     shipmentmoveports.StopActualActionDepart,
		OccurredAt: &departed,
		Source:     trackingevent.SourceDispatcher,
		SourceKey:  "idem-2",
	})
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeApplied, result.Outcome)

	replay, err := h.tracking.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
		StopID:     h.move.Stops[0].ID,
		Action:     shipmentmoveports.StopActualActionDepart,
		OccurredAt: &departed,
		Source:     trackingevent.SourceDispatcher,
		SourceKey:  "idem-2",
	})
	require.NoError(t, err)
	assert.True(t, replay.Replayed)
	assert.Equal(t, departed, *h.persisted(t, ctx).ActiveStopsBySequence()[0].ActualDeparture)
}

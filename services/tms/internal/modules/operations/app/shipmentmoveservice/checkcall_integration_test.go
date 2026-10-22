//go:build integration

package shipmentmoveservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/carrierassignmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/checkcallrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/trackingeventrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/checkcallservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/trackingeventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
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

const secondsPerHourTest = int64(3600)

type fakeOrganizations struct {
	tenantports.OrganizationRepository
}

func (fakeOrganizations) GetByID(
	_ context.Context,
	req tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{ID: req.TenantInfo.OrgID, Name: "Acme Brokerage"}, nil
}

type checkCallHarness struct {
	svc        *checkcallservice.Service
	links      checkcallports.TrackingLinkRepository
	controls   dispatchcontrolports.DispatchControlRepository
	moveRepo   shipmentports.ShipmentMoveRepository
	db         *bun.DB
	tenantInfo pagination.TenantInfo
	move       *shipment.ShipmentMove
	customerID pulid.ID
}

func newCheckCallHarness(t *testing.T) (context.Context, *checkCallHarness) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	moveSvc, shipmentRepo, moveRepo, tenantInfo, fixture, controlRepo := newIntegrationService(
		t,
		ctx,
		db,
	)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.AnythingOfType("tenantports.GetShipmentControlRequest")).
		Return(&tenant.ShipmentControl{}, nil).
		Maybe()
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo, testutil.ShipmentGraphParams{
		BOL:          "BOL-CHECKCALL",
		ProNumber:    "PRO-CHECKCALL",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusAssigned},
	})
	move := graph.Moves[0]

	carrierID := pulid.MustNew("car_")
	_, err := db.NewRaw(
		`INSERT INTO carriers (id, business_unit_id, organization_id, code, name)
		VALUES (?, ?, ?, 'REDLINE', 'Redline Freight')`,
		carrierID, tenantInfo.BuID, tenantInfo.OrgID,
	).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(
		`INSERT INTO carrier_assignments (
			id, business_unit_id, organization_id, shipment_move_id, carrier_id, status,
			base_rate, base_amount, total_cost, confirmed_at
		) VALUES (?, ?, ?, ?, ?, 'Confirmed', 1000, 1000, 1000, ?)`,
		pulid.MustNew("casn_"), tenantInfo.BuID, tenantInfo.OrgID, move.ID, carrierID,
		timeutils.NowUnix()-10*secondsPerHourTest,
	).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(
		`UPDATE shipment_moves SET coverage_type = 'carrier' WHERE id = ?`,
		move.ID,
	).Exec(ctx)
	require.NoError(t, err)

	conn := postgres.NewTestConnection(db)
	tracking := trackingeventservice.New(trackingeventservice.Params{
		DB:     conn,
		Logger: zap.NewNop(),
		Repo: trackingeventrepository.New(
			trackingeventrepository.Params{DB: conn, Logger: zap.NewNop()},
		),
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
		MoveService:  moveSvc,
	})
	links := checkcallrepository.NewTrackingLinks(
		checkcallrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	controls := dispatchcontrolrepository.New(
		dispatchcontrolrepository.Params{DB: conn, Logger: zap.NewNop()},
	)
	svc := checkcallservice.New(checkcallservice.Params{
		DB:     conn,
		Logger: zap.NewNop(),
		Config: &config.Config{
			Tendering: config.TenderingConfig{PublicBaseURL: "https://app.example.com"},
		},
		Repo:         checkcallrepository.New(checkcallrepository.Params{DB: conn, Logger: zap.NewNop()}),
		Links:        links,
		MoveRepo:     moveRepo,
		ShipmentRepo: shipmentRepo,
		CarrierAssignments: carrierassignmentrepository.New(
			carrierassignmentrepository.Params{DB: conn, Logger: zap.NewNop()},
		),
		CarrierRepo: carrierrepository.New(
			carrierrepository.Params{DB: conn, Logger: zap.NewNop()},
		),
		OrgRepo:             fakeOrganizations{},
		DispatchControlRepo: controls,
		TrackingEvents:      tracking,
	})

	return ctx, &checkCallHarness{
		svc:        svc,
		links:      links,
		controls:   controls,
		moveRepo:   moveRepo,
		db:         db,
		tenantInfo: tenantInfo,
		move:       move,
		customerID: fixture.Customer.ID,
	}
}

func (h *checkCallHarness) issue(t *testing.T, ctx context.Context) string {
	t.Helper()
	issued, err := h.svc.IssueLink(ctx, &checkcallports.IssueTrackingLinkRequest{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
		IssuedByID: h.tenantInfo.UserID,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(issued.URL, "https://app.example.com/carrier-tracking/"))
	return strings.TrimPrefix(issued.URL, "https://app.example.com/carrier-tracking/")
}

func (h *checkCallHarness) persisted(t *testing.T, ctx context.Context) *shipment.ShipmentMove {
	t.Helper()
	move, err := h.moveRepo.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
		MoveID:            h.move.ID,
		TenantInfo:        h.tenantInfo,
		ExpandMoveDetails: true,
	})
	require.NoError(t, err)
	return move
}

func (h *checkCallHarness) enableMonitoring(t *testing.T, ctx context.Context, hours int16) {
	t.Helper()
	control, err := h.controls.GetOrCreate(ctx, h.tenantInfo.OrgID, h.tenantInfo.BuID)
	require.NoError(t, err)
	control.EnableCheckCallMonitoring = true
	control.CheckCallIntervalHours = hours
	_, err = h.controls.Update(ctx, control)
	require.NoError(t, err)
}

func TestCheckCallIntegrationCarrierPostSettlesTheStop(t *testing.T) {
	ctx, h := newCheckCallHarness(t)
	token := h.issue(t, ctx)

	view, err := h.svc.PreviewByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "Redline Freight", view.CarrierName)
	assert.Equal(t, "Acme Brokerage", view.CompanyName)
	require.Len(t, view.Stops, len(h.move.Stops))
	assert.False(t, view.Closed)

	arrivedAt := timeutils.NowUnix() - secondsPerHourTest
	first := view.Stops[0]
	result, err := h.svc.PostByToken(ctx, token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     first.ID,
			ReportedAt: &arrivedAt,
			City:       "Dallas",
			StateCode:  "tx",
		},
		ClientEventID: "tap-1",
	})
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeApplied, result.Outcome)
	assert.False(t, result.Replayed)

	move := h.persisted(t, ctx)
	stop := move.ActiveStopsBySequence()[0]
	require.NotNil(t, stop.ActualArrival)
	assert.Equal(t, arrivedAt, *stop.ActualArrival)
	assert.Equal(t, shipment.MoveStatusInTransit, move.Status)

	var events []struct {
		Source       string `bun:"source"`
		RawReference string `bun:"raw_reference"`
	}
	require.NoError(t, h.db.NewRaw(
		`SELECT source, raw_reference FROM tracking_events WHERE shipment_move_id = ?`,
		h.move.ID,
	).Scan(dbscope.WithTenant(ctx, h.tenantInfo.DBTenant()), &events))
	require.Len(t, events, 1)
	assert.Equal(t, string(trackingevent.SourceCarrier), events[0].Source)
	assert.True(t, strings.HasPrefix(events[0].RawReference, "carrier_tracking_link:"))

	again, err := h.svc.PostByToken(ctx, token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     first.ID,
			ReportedAt: &arrivedAt,
		},
		ClientEventID: "tap-1",
	})
	require.NoError(t, err)
	assert.True(t, again.Replayed)

	calls, err := h.svc.ListForShipment(ctx, &checkcallports.ListCheckCallsByShipmentRequest{
		TenantInfo: h.tenantInfo,
		ShipmentID: h.move.ShipmentID,
	})
	require.NoError(t, err)
	require.Len(t, calls, 1, "the repeated tap is stored once")
	assert.Equal(t, checkcall.SourceCarrier, calls[0].Source)
	assert.Equal(t, "TX", calls[0].StateCode)
	assert.False(t, calls[0].TrackingEventID.IsNil())
	assert.False(t, calls[0].TrackingLinkID.IsNil())

	link, err := h.svc.ActiveLink(ctx, h.tenantInfo, h.move.ID)
	require.NoError(t, err)
	require.NotNil(t, link.LastUsedAt, "a post marks the link used")
}

func TestCheckCallIntegrationAQuietLoadGoesOverdue(t *testing.T) {
	ctx, h := newCheckCallHarness(t)
	token := h.issue(t, ctx)
	view, err := h.svc.PreviewByToken(ctx, token)
	require.NoError(t, err)

	arrivedAt := timeutils.NowUnix() - 2*secondsPerHourTest
	_, err = h.svc.PostByToken(ctx, token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     view.Stops[0].ID,
			ReportedAt: &arrivedAt,
		},
		ClientEventID: "tap-arrive",
	})
	require.NoError(t, err)

	now := timeutils.NowUnix()
	overdue, err := h.svc.ListOverdue(ctx, h.tenantInfo, now+5*secondsPerHourTest)
	require.NoError(t, err)
	assert.Empty(t, overdue, "nothing is overdue while monitoring is off")

	h.enableMonitoring(t, ctx, 1)
	tenants, err := h.controls.ListCheckCallMonitoringTenants(ctx)
	require.NoError(t, err)
	assert.Contains(t, tenants, pagination.TenantInfo{
		OrgID: h.tenantInfo.OrgID,
		BuID:  h.tenantInfo.BuID,
	})

	overdue, err = h.svc.ListOverdue(ctx, h.tenantInfo, now)
	require.NoError(t, err)
	assert.Empty(t, overdue, "the carrier just checked in")

	overdue, err = h.svc.ListOverdue(ctx, h.tenantInfo, now+2*secondsPerHourTest)
	require.NoError(t, err)
	require.Len(t, overdue, 1)
	assert.Equal(t, h.move.ID, overdue[0].MoveID)
	assert.Equal(t, "Redline Freight", overdue[0].CarrierName)
	assert.Equal(t, int16(1), overdue[0].IntervalHours)
	assert.Positive(t, overdue[0].OverdueBy)

	_, err = h.db.NewRaw(
		`UPDATE customers SET check_call_interval_hours = 4 WHERE id = ?`,
		h.customerID,
	).Exec(dbscope.WithTenant(ctx, h.tenantInfo.DBTenant()))
	require.NoError(t, err)

	overdue, err = h.svc.ListOverdue(ctx, h.tenantInfo, now+2*secondsPerHourTest)
	require.NoError(t, err)
	assert.Empty(t, overdue, "the customer's own window is longer")

	overdue, err = h.svc.ListOverdue(ctx, h.tenantInfo, now+5*secondsPerHourTest)
	require.NoError(t, err)
	require.Len(t, overdue, 1)
	assert.Equal(t, int16(4), overdue[0].IntervalHours)

	logged, err := h.svc.Log(ctx, &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceDispatcher,
		Details: checkcallports.CheckCallDetails{
			Kind:        checkcall.KindUpdate,
			City:        "Tulsa",
			Delayed:     true,
			DelayReason: "Weather on I-44",
		},
	})
	require.NoError(t, err)
	assert.True(t, logged.CheckCall.TrackingEventID.IsNil())

	overdue, err = h.svc.ListOverdue(ctx, h.tenantInfo, now+3*secondsPerHourTest)
	require.NoError(t, err)
	assert.Empty(t, overdue, "a dispatcher's call resets the clock")
}

func TestCheckCallIntegrationARevokedLinkIsDeadAndPurged(t *testing.T) {
	ctx, h := newCheckCallHarness(t)
	first := h.issue(t, ctx)
	second := h.issue(t, ctx)

	_, err := h.svc.PreviewByToken(ctx, first)
	require.Error(t, err, "a reissued link replaces the old one")
	assert.True(t, errortypes.IsError(err))

	_, err = h.svc.PreviewByToken(ctx, second)
	require.NoError(t, err)

	require.NoError(t, h.svc.RevokeLink(ctx, &checkcallports.RevokeTrackingLinkRequest{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
	}))
	_, err = h.svc.PreviewByToken(ctx, second)
	require.Error(t, err)

	err = h.svc.RevokeLink(ctx, &checkcallports.RevokeTrackingLinkRequest{
		TenantInfo: h.tenantInfo,
		MoveID:     h.move.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	purged, err := h.links.PurgeDead(ctx, timeutils.NowUnix()+1, 100)
	require.NoError(t, err)
	assert.Equal(t, 2, purged)
}

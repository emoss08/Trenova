package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const testDay = int64(86400)

type testDBConnection struct{}

func (testDBConnection) DB() *bun.DB                          { return nil }
func (testDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (testDBConnection) HealthCheck(context.Context) error    { return nil }
func (testDBConnection) IsHealthy(context.Context) bool       { return true }
func (testDBConnection) Close() error                         { return nil }
func (testDBConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeTenderRepo struct {
	repositories.TenderRepository

	liveTender *tender.Tender
	created    *tender.Tender
	workflowID string
}

func (f *fakeTenderRepo) GetLiveByMoveID(
	_ context.Context,
	_ repositories.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return f.liveTender, nil
}

func (f *fakeTenderRepo) Create(
	_ context.Context,
	entity *tender.Tender,
) (*tender.Tender, error) {
	entity.ID = pulid.MustNew("ten_")
	f.created = entity
	return entity, nil
}

func (f *fakeTenderRepo) SetWorkflowID(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	workflowID string,
) error {
	f.workflowID = workflowID
	return nil
}

type fakeGuideRepo struct {
	repositories.RoutingGuideRepository

	matched *tender.RoutingGuide
}

func (f *fakeGuideRepo) MatchLane(
	_ context.Context,
	_ *repositories.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	return f.matched, nil
}

type fakeWorkflowStarter struct {
	portservices.WorkflowStarter

	started []client.StartWorkflowOptions
	err     error
}

func (f *fakeWorkflowStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.started = append(f.started, options)
	return nil, nil
}

type recordingEventService struct {
	events []*portservices.RecordShipmentEventParams
}

func (r *recordingEventService) Record(
	_ context.Context,
	params *portservices.RecordShipmentEventParams,
) error {
	r.events = append(r.events, params)
	return nil
}

func (r *recordingEventService) List(
	_ context.Context,
	_ *repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, nil
}

func (r *recordingEventService) eventsOfType(
	eventType shipmentevent.Type,
) []*portservices.RecordShipmentEventParams {
	matched := make([]*portservices.RecordShipmentEventParams, 0, len(r.events))
	for _, event := range r.events {
		if event.Type == eventType {
			matched = append(matched, event)
		}
	}
	return matched
}

type createTestDeps struct {
	tenderRepo            *fakeTenderRepo
	guideRepo             *fakeGuideRepo
	carrierRepo           *mocks.MockCarrierRepository
	assignmentRepo        *mocks.MockAssignmentRepository
	carrierAssignmentRepo *mocks.MockCarrierAssignmentRepository
	shipmentRepo          *mocks.MockShipmentRepository
	holdRepo              *mocks.MockShipmentHoldRepository
	workflows             *fakeWorkflowStarter
	events                *recordingEventService
	notifRepo             *mocks.MockNotificationRepository
	audit                 *recordingAuditService
	svc                   *Service
}

func setupCreateTest(t *testing.T) *createTestDeps {
	t.Helper()

	deps := &createTestDeps{
		tenderRepo:            &fakeTenderRepo{},
		guideRepo:             &fakeGuideRepo{},
		carrierRepo:           mocks.NewMockCarrierRepository(t),
		assignmentRepo:        mocks.NewMockAssignmentRepository(t),
		carrierAssignmentRepo: mocks.NewMockCarrierAssignmentRepository(t),
		shipmentRepo:          mocks.NewMockShipmentRepository(t),
		holdRepo:              mocks.NewMockShipmentHoldRepository(t),
		workflows:             &fakeWorkflowStarter{},
		events:                &recordingEventService{},
		notifRepo:             mocks.NewMockNotificationRepository(t),
		audit:                 &recordingAuditService{},
	}

	deps.svc = &Service{
		l:                     zap.NewNop(),
		db:                    testDBConnection{},
		repo:                  deps.tenderRepo,
		guideRepo:             deps.guideRepo,
		carrierRepo:           deps.carrierRepo,
		carrierAssignmentRepo: deps.carrierAssignmentRepo,
		assignmentRepo:        deps.assignmentRepo,
		shipmentRepo:          deps.shipmentRepo,
		holdRepo:              deps.holdRepo,
		workflows:             deps.workflows,
		eventService:          deps.events,
		auditService:          deps.audit,
		notifications: notificationservice.New(notificationservice.Params{
			Logger:   zap.NewNop(),
			Repo:     deps.notifRepo,
			Realtime: &mocks.NoopRealtimeService{},
		}),
	}

	return deps
}

func createTestTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func tenderableMove() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:         pulid.MustNew("smv_"),
		ShipmentID: pulid.MustNew("shp_"),
		Status:     shipment.MoveStatusNew,
	}
}

func expectTenderableMove(
	deps *createTestDeps,
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
) {
	deps.assignmentRepo.On("GetMoveByID", mock.Anything, tenantInfo, move.ID).
		Return(move, nil)
	deps.holdRepo.On("HasActiveDispatchHold", mock.Anything, mock.Anything).
		Return(false, nil)
	deps.assignmentRepo.On("GetByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil)
	deps.carrierAssignmentRepo.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil)
	deps.shipmentRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: move.ShipmentID}, nil)
}

func eligibleCarrier(name string) *carrier.Carrier {
	now := timeutils.NowUnix()
	return &carrier.Carrier{
		ID:               pulid.MustNew("car_"),
		Status:           carrier.StatusActive,
		ComplianceStatus: carrier.ComplianceStatusQualified,
		Name:             name,
		Email:            "dispatch@" + name + ".test",
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{
			{
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "AUTO-1",
				ExpirationDate: now + 365*testDay,
			},
			{
				PolicyType:     carrier.InsurancePolicyTypeCargoLiability,
				PolicyNumber:   "CARGO-1",
				ExpirationDate: now + 365*testDay,
			},
		},
	}
}

func blockedCarrier(name string) *carrier.Carrier {
	entity := eligibleCarrier(name)
	entity.Status = carrier.StatusDoNotUse
	return entity
}

func warnedCarrier(name string) *carrier.Carrier {
	entity := eligibleCarrier(name)
	entity.InsurancePolicies[0].ExpirationDate = timeutils.NowUnix() + 10*testDay
	return entity
}

func spotLine(carrierID pulid.ID) SpotTenderLine {
	return SpotTenderLine{
		CarrierID:  carrierID,
		RateMethod: shipment.CarrierRateMethodFlat,
		Rate:       decimal.NewFromInt(1500),
		Channel:    tender.ChannelEmail,
	}
}

func guideEntry(carrierID pulid.ID, rank int16) *tender.RoutingGuideEntry {
	return &tender.RoutingGuideEntry{
		ID:              pulid.MustNew("rge_"),
		CarrierID:       carrierID,
		Rank:            rank,
		RateMethod:      shipment.CarrierRateMethodFlat,
		Rate:            decimal.NewFromInt(1200),
		OfferTTLSeconds: tender.DefaultOfferTTLSeconds,
		Channel:         tender.ChannelEmail,
	}
}

func fieldError(multiErr *errortypes.MultiError, field string) *errortypes.Error {
	for _, err := range multiErr.Errors {
		if err.Field == field {
			return err
		}
	}
	return nil
}

func TestCreateSpotBlockedCarrierFailsPerLine(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	healthy := eligibleCarrier("acme")
	blocked := blockedCarrier("lapsed")
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			req, ok := args.Get(1).(repositories.GetCarriersByIDsRequest)
			require.True(t, ok)
			assert.True(t, req.IncludeInsurancePolicies)
		}).
		Return([]*carrier.Carrier{healthy, blocked}, nil)

	created, err := deps.svc.CreateSpot(t.Context(), &CreateSpotTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
		Mode:           tender.ModeSpotBroadcast,
		Lines:          []SpotTenderLine{spotLine(healthy.ID), spotLine(blocked.ID)},
	})

	require.Nil(t, created)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	lineErr := fieldError(multiErr, "lines[1].carrierId")
	require.NotNil(t, lineErr)
	assert.Contains(t, lineErr.Message, "Carrier is not eligible for tendering")
	assert.Contains(t, lineErr.Message, "Carrier status is DoNotUse")
	assert.Nil(t, fieldError(multiErr, "lines[0].carrierId"))
	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
}

func TestCreateSpotWarningWithoutOverrideIsOverridable(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	warned := warnedCarrier("expiring")
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{warned}, nil)

	created, err := deps.svc.CreateSpot(t.Context(), &CreateSpotTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
		Mode:           tender.ModeSpotBroadcast,
		Lines:          []SpotTenderLine{spotLine(warned.ID)},
	})

	require.Nil(t, created)
	require.Error(t, err)
	var bizErr *errortypes.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, "true", bizErr.Params["overridable"])
	assert.Contains(t, bizErr.Message, "insurance warnings")
	assert.Contains(t, bizErr.Message, "expiring")
	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
}

func TestCreateSpotWarningWithOverrideProceeds(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	warned := warnedCarrier("expiring")
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{warned}, nil)

	created, err := deps.svc.CreateSpot(t.Context(), &CreateSpotTenderRequest{
		TenantInfo:                tenantInfo,
		ShipmentMoveID:            move.ID,
		Mode:                      tender.ModeSpotBroadcast,
		Lines:                     []SpotTenderLine{spotLine(warned.ID)},
		OverrideInsuranceWarnings: true,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Len(t, created.Offers, 1)
	assert.Equal(t, warned.ID, created.Offers[0].CarrierID)
	require.Len(t, deps.workflows.started, 1)
	assert.Equal(t, deps.workflows.started[0].ID, deps.tenderRepo.workflowID)
}

func TestCreateWaterfallSkipsBlockedEntriesWithAudit(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	blocked := blockedCarrier("lapsed")
	warned := warnedCarrier("expiring")
	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:     pulid.MustNew("rgd_"),
		Status: "Active",
		Entries: []*tender.RoutingGuideEntry{
			guideEntry(blocked.ID, 1),
			guideEntry(warned.ID, 2),
			guideEntry(healthy.ID, 3),
		},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			req, ok := args.Get(1).(repositories.GetCarriersByIDsRequest)
			require.True(t, ok)
			assert.True(t, req.IncludeInsurancePolicies)
			assert.Len(t, req.CarrierIDs, 3)
		}).
		Return([]*carrier.Carrier{blocked, warned, healthy}, nil)

	var notified *notification.Notification
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*notification.Notification)
			require.True(t, ok)
			notified = entity
		}).
		Return(&notification.Notification{}, nil)

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Len(t, created.Offers, 2)
	assert.Equal(t, warned.ID, created.Offers[0].CarrierID)
	assert.Equal(t, healthy.ID, created.Offers[1].CarrierID)

	skips := deps.events.eventsOfType(shipmentevent.TypeTenderEntrySkipped)
	require.Len(t, skips, 1)
	assert.Equal(t, created.ShipmentID, skips[0].ShipmentID)
	assert.Equal(t, "lapsed", skips[0].Metadata["carrierName"])

	warns := deps.events.eventsOfType(shipmentevent.TypeTenderEntryWarned)
	require.Len(t, warns, 1)
	assert.Equal(t, "expiring", warns[0].Metadata["carrierName"])

	require.NotNil(t, created.Screening)
	require.Len(t, created.Screening.Skipped, 1)
	assert.Equal(t, "lapsed", created.Screening.Skipped[0].CarrierName)
	assert.Equal(t, int16(1), created.Screening.Skipped[0].Rank)
	assert.NotEmpty(t, created.Screening.Skipped[0].Reasons)
	require.Len(t, created.Screening.Warned, 1)
	assert.Equal(t, "expiring", created.Screening.Warned[0].CarrierName)
	assert.Equal(t, int16(2), created.Screening.Warned[0].Rank)

	require.NotNil(t, notified)
	assert.Equal(t, "tender_entries_skipped", notified.EventType)
	assert.Contains(t, notified.Message, "1 of 3")
	assert.Contains(t, notified.Message, "lapsed")
	assert.Equal(t, created.ID.String(), *notified.CorrelationID)
}

func TestCreateWaterfallAllBlockedFails(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	first := blockedCarrier("lapsed")
	second := blockedCarrier("inactive")
	guideID := pulid.MustNew("rgd_")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:     guideID,
		Status: "Active",
		Entries: []*tender.RoutingGuideEntry{
			guideEntry(first.ID, 1),
			guideEntry(second.ID, 2),
		},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{first, second}, nil)

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.Nil(t, created)
	require.Error(t, err)
	var bizErr *errortypes.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Contains(t, bizErr.Message, "No eligible carriers on the routing guide")
	assert.Equal(t, guideID.String(), bizErr.Params["routingGuideId"])
	assert.Nil(t, deps.tenderRepo.created)
	assert.Empty(t, deps.workflows.started)
	assert.Empty(t, deps.events.events)
}

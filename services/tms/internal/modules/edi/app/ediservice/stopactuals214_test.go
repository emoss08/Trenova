package ediservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmenteventportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/trackingeventportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stopActuals214Fixture struct {
	service    *Service
	moveRepo   *shipmentportsmock.MockShipmentMoveRepository
	tracking   *trackingeventportsmock.MockTrackingEventService
	tenantInfo pagination.TenantInfo
	moveID     pulid.ID
	origin     *shipment.Stop
	middle     *shipment.Stop
	final      *shipment.Stop
	recorded   []*trackingeventports.RecordObservedStopEventParams
}

func newStopActuals214Fixture(t *testing.T) *stopActuals214Fixture {
	t.Helper()

	commentRepo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	eventRepo := shipmenteventportsmock.NewMockShipmentEventRepository(t)
	moveRepo := shipmentportsmock.NewMockShipmentMoveRepository(t)
	tracking := trackingeventportsmock.NewMockTrackingEventService(t)

	userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).
		Maybe()
	commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Maybe()
	eventRepo.EXPECT().
		Insert(mock.Anything, mock.AnythingOfType("*shipmentevent.Event")).
		Return(nil).
		Maybe()

	fixture := &stopActuals214Fixture{
		moveRepo: moveRepo,
		tracking: tracking,
		moveID:   pulid.MustNew("sm_"),
		tenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
	fixture.origin = &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypePickup,
		Status:   shipment.StopStatusNew,
		Sequence: 0,
	}
	fixture.middle = &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 1,
	}
	fixture.final = &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 2,
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
		shipmentCommentRepo: commentRepo,
		userRepo:            userRepo,
		shipmentEventRepo:   eventRepo,
		shipmentMoveRepo:    moveRepo,
		trackingEvents:      tracking,
	}
	return fixture
}

func (f *stopActuals214Fixture) move() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:     f.moveID,
		Status: shipment.MoveStatusAssigned,
		Stops:  []*shipment.Stop{f.origin, f.middle, f.final},
	}
}

func (f *stopActuals214Fixture) expectMoveLoad() {
	f.moveRepo.EXPECT().
		GetByID(mock.MatchedBy(func(context.Context) bool { return true }), mock.MatchedBy(
			func(req *shipmentports.GetMoveByIDRequest) bool {
				return req.MoveID == f.moveID &&
					req.TenantInfo == f.tenantInfo &&
					req.ExpandMoveDetails &&
					!req.ForUpdate
			},
		)).
		Return(f.move(), nil).
		Once()
}

func (f *stopActuals214Fixture) expectRecordStopActual(times int) {
	f.expectOutcome(times, trackingevent.OutcomeApplied, "")
}

func (f *stopActuals214Fixture) expectOutcome(
	times int,
	outcome trackingevent.Outcome,
	reason string,
) {
	f.tracking.EXPECT().
		RecordObserved(mock.Anything, mock.AnythingOfType("*trackingeventports.RecordObservedStopEventParams")).
		RunAndReturn(
			func(
				_ context.Context,
				params *trackingeventports.RecordObservedStopEventParams,
			) (*trackingeventports.StopEventResult, error) {
				f.recorded = append(f.recorded, params)
				return &trackingeventports.StopEventResult{
					Move:    f.move(),
					Outcome: outcome,
					Reason:  reason,
				}, nil
			},
		).
		Times(times)
}

func (f *stopActuals214Fixture) request(
	statusCode string,
	eventAt int64,
) *RecordTenderedShipmentStatusRequest {
	offer := acceptedTenderOfferForTest()
	offer.Tender.ShipmentMoveID = f.moveID
	return &RecordTenderedShipmentStatusRequest{
		TenantInfo: f.tenantInfo,
		Offer:      offer,
		StatusCode: statusCode,
		EventAt:    eventAt,
	}
}

func TestRecordTenderedShipmentStatus_X3ArrivesAtFirstOriginStop(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X3", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 1)
	recorded := fixture.recorded[0]
	require.Equal(t, shipment.VisitArrival, recorded.Kind)
	require.Equal(t, fixture.origin.ID, recorded.StopID)
	require.Equal(t, fixture.moveID, recorded.MoveID)
	require.Equal(t, fixture.tenantInfo, recorded.TenantInfo)
	require.Equal(t, trackingevent.SourceEDI, recorded.Source)
	require.Equal(t, trackingevent.MatchStopRole, recorded.MatchMethod)
	require.Equal(t, "X3", recorded.SourceStatus)
	require.EqualValues(t, 1767787200, recorded.EventAt)
	require.Contains(t, recorded.SourceKey, fixture.origin.ID.String())
}

func TestRecordTenderedShipmentStatus_AFDepartsOriginStop(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("AF", 0),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 1)
	recorded := fixture.recorded[0]
	require.Equal(t, shipment.VisitDeparture, recorded.Kind)
	require.Equal(t, fixture.origin.ID, recorded.StopID)
	require.Positive(t, recorded.EventAt)
}

func TestRecordTenderedShipmentStatus_X1ArrivesAtTheNextDeliveryStop(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X1", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 1)
	require.Equal(t, shipment.VisitArrival, fixture.recorded[0].Kind)
	require.Equal(t, fixture.middle.ID, fixture.recorded[0].StopID)
}

func TestRecordTenderedShipmentStatus_D1ArrivesThenDepartsWhenArrivalMissing(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(2)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("D1", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 2)
	require.Equal(t, shipment.VisitArrival, fixture.recorded[0].Kind)
	require.Equal(t, trackingevent.MatchInferred, fixture.recorded[0].MatchMethod)
	require.Equal(t, shipment.VisitDeparture, fixture.recorded[1].Kind)
	require.Equal(t, fixture.middle.ID, fixture.recorded[0].StopID)
	require.Equal(t, fixture.middle.ID, fixture.recorded[1].StopID)
	require.Equal(t, fixture.recorded[1].SourceKey+":arrival", fixture.recorded[0].SourceKey)
	require.EqualValues(t, 1767787200, fixture.recorded[1].EventAt)
}

func TestRecordTenderedShipmentStatus_D1OnlyDepartsWhenAlreadyArrived(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	arrivedAt := int64(1767780000)
	fixture.middle.ActualArrival = &arrivedAt
	fixture.middle.Status = shipment.StopStatusInTransit
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("D1", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 1)
	require.Equal(t, shipment.VisitDeparture, fixture.recorded[0].Kind)
	require.Equal(t, fixture.middle.ID, fixture.recorded[0].StopID)
}

func TestRecordTenderedShipmentStatus_UnmappedCodeStaysCommentOnly(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X6", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Empty(t, fixture.recorded)
	fixture.moveRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	fixture.tracking.AssertNotCalled(t, "RecordObserved", mock.Anything, mock.Anything)
}

func TestRecordTenderedShipmentStatus_StopActualBusinessErrorDegradesToWarning(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.tracking.EXPECT().
		RecordObserved(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewBusinessError("This load has been canceled")).
		Once()

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X1", 1767787200),
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "X1")
	require.Contains(t, warnings[0], "This load has been canceled")
}

func TestRecordTenderedShipmentStatus_StopActualInfrastructureErrorPropagates(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.tracking.EXPECT().
		RecordObserved(mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable")).
		Once()

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("AF", 0),
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "database unavailable")
	require.Empty(t, warnings)
}

func TestRecordTenderedShipmentStatus_MoveNotFoundDegradesToWarning(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.moveRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetMoveByIDRequest")).
		Return(nil, errortypes.NewNotFoundError("Shipment move")).
		Once()

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X3", 0),
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "could not resolve the tendered move")
}

func TestRecordTenderedShipmentStatus_MissingMoveServiceStaysCommentOnly(t *testing.T) {
	t.Parallel()

	commentRepo := shipmentportsmock.NewMockShipmentCommentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	service := &Service{
		l:                   zap.NewNop(),
		shipmentCommentRepo: commentRepo,
		userRepo:            userRepo,
	}
	userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).
		Once()
	commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()

	offer := acceptedTenderOfferForTest()
	offer.Tender.ShipmentMoveID = pulid.MustNew("sm_")
	warnings, err := service.RecordTenderedShipmentStatus(
		t.Context(),
		&RecordTenderedShipmentStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Offer:      offer,
			StatusCode: "X3",
		},
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
}

func TestStopActualsPlanForStatusCode(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		code            string
		mapped          bool
		target          stopActualTarget
		action          shipmentports.StopActualAction
		arriveIfMissing bool
	}{
		{
			code:   "X3",
			mapped: true,
			target: stopTargetOrigin,
			action: shipmentports.StopActualActionArrive,
		},
		{
			code:   "AF",
			mapped: true,
			target: stopTargetOrigin,
			action: shipmentports.StopActualActionDepart,
		},
		{
			code:   "X1",
			mapped: true,
			target: stopTargetDestination,
			action: shipmentports.StopActualActionArrive,
		},
		{
			code:            "D1",
			mapped:          true,
			target:          stopTargetDestination,
			action:          shipmentports.StopActualActionDepart,
			arriveIfMissing: true,
		},
		{
			code:   "CD",
			mapped: true,
			target: stopTargetDestination,
			action: shipmentports.StopActualActionDepart,
		},
		{code: "A3"},
		{code: "SD"},
		{code: "A7"},
		{code: "X6"},
		{code: ""},
	} {
		plan, ok := stopActualsPlanForStatusCode(testCase.code)
		require.Equal(t, testCase.mapped, ok, testCase.code)
		if testCase.mapped {
			require.Equal(t, testCase.target, plan.target, testCase.code)
			require.Equal(t, testCase.action, plan.action, testCase.code)
			require.Equal(t, testCase.arriveIfMissing, plan.arriveIfMissing, testCase.code)
		}
	}
}

func placedStop(stop *shipment.Stop, city, state, timezone string) {
	stop.Location = &location.Location{
		City:     city,
		Timezone: timezone,
		State:    &usstate.UsState{Abbreviation: state},
	}
}

func TestRecordTenderedShipmentStatus_StopSequenceReferenceWins(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)
	sequence := fixture.final.Sequence
	req := fixture.request("X1", 1767787200)
	req.Event = &edi.ShipmentStatusEvent{StatusCode: "X1", StopSequence: &sequence}

	warnings, err := fixture.service.RecordTenderedShipmentStatus(t.Context(), req)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, fixture.final.ID, fixture.recorded[0].StopID)
	require.Equal(t, trackingevent.MatchStopReference, fixture.recorded[0].MatchMethod)
}

func TestRecordTenderedShipmentStatus_PlaceNamesTheStop(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	placedStop(fixture.middle, "Fort Wayne", "IN", "")
	placedStop(fixture.final, "Dayton", "OH", "")
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)
	missing := int64(42)
	latitude, longitude := 39.7589, -84.1916
	req := fixture.request("X1", 1767787200)
	req.Event = &edi.ShipmentStatusEvent{
		StatusCode:   "X1",
		StopSequence: &missing,
		City:         "DAYTON",
		StateCode:    "oh",
		Latitude:     &latitude,
		Longitude:    &longitude,
	}
	req.RawReference = "edi_inbound_file:f:0001:2"

	warnings, err := fixture.service.RecordTenderedShipmentStatus(t.Context(), req)

	require.NoError(t, err)
	require.Empty(t, warnings)
	recorded := fixture.recorded[0]
	require.Equal(t, fixture.final.ID, recorded.StopID)
	require.Equal(t, trackingevent.MatchLocation, recorded.MatchMethod)
	require.Equal(t, &latitude, recorded.Latitude)
	require.Equal(t, &longitude, recorded.Longitude)
	require.Equal(t, "edi_inbound_file:f:0001:2", recorded.RawReference)
}

func TestRecordTenderedShipmentStatus_LocalTimeIsReadInTheStopZone(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	placedStop(fixture.origin, "Joliet", "IL", "America/Chicago")
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	req := fixture.request("X3", time.Date(2026, 1, 7, 8, 0, 0, 0, time.UTC).Unix())
	req.Event = &edi.ShipmentStatusEvent{
		StatusCode: "X3",
		DateTime:   edi.X12DateTime{Date: "20260107", Time: "0800", TimeCode: "LT"},
	}

	_, err = fixture.service.RecordTenderedShipmentStatus(t.Context(), req)

	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 1, 7, 8, 0, 0, 0, chicago).Unix(), fixture.recorded[0].EventAt)
}

func TestRecordTenderedShipmentStatus_ZonedTimeIgnoresTheStopZone(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	placedStop(fixture.origin, "Joliet", "IL", "America/Chicago")
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)
	eventAt := time.Date(2026, 1, 7, 13, 0, 0, 0, time.UTC).Unix()
	req := fixture.request("X3", eventAt)
	req.Event = &edi.ShipmentStatusEvent{
		StatusCode: "X3",
		DateTime:   edi.X12DateTime{Date: "20260107", Time: "0800", TimeCode: "ET"},
	}

	_, err := fixture.service.RecordTenderedShipmentStatus(t.Context(), req)

	require.NoError(t, err)
	require.Equal(t, eventAt, fixture.recorded[0].EventAt)
}

func TestRecordTenderedShipmentStatus_RefusedAndWaitingOutcomesAreWarnings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		outcome trackingevent.Outcome
		verb    string
	}{
		{outcome: trackingevent.OutcomeRefused, verb: "was not applied"},
		{outcome: trackingevent.OutcomePending, verb: "is waiting"},
	} {
		fixture := newStopActuals214Fixture(t)
		fixture.expectMoveLoad()
		fixture.expectOutcome(1, tc.outcome, "Arrival is before the previous stop's departure")

		warnings, err := fixture.service.RecordTenderedShipmentStatus(
			t.Context(),
			fixture.request("X1", 1767787200),
		)

		require.NoError(t, err)
		require.Len(t, warnings, 1)
		require.Contains(t, warnings[0], "X1 arrival at stop 1 "+tc.verb)
		require.Contains(t, warnings[0], "Arrival is before the previous stop's departure")
	}
}

func TestRecordTenderedShipmentStatus_SupersededAndDuplicateAreQuiet(t *testing.T) {
	t.Parallel()

	for _, outcome := range []trackingevent.Outcome{
		trackingevent.OutcomeSuperseded,
		trackingevent.OutcomeDuplicate,
	} {
		fixture := newStopActuals214Fixture(t)
		fixture.expectMoveLoad()
		fixture.expectOutcome(1, outcome, "Another report was used")

		warnings, err := fixture.service.RecordTenderedShipmentStatus(
			t.Context(),
			fixture.request("AF", 1767787200),
		)

		require.NoError(t, err)
		require.Empty(t, warnings)
	}
}

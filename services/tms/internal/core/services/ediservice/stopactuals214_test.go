package ediservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stopActuals214Fixture struct {
	service    *Service
	moveRepo   *mocks.MockShipmentMoveRepository
	moveSvc    *mocks.MockShipmentMoveService
	tenantInfo pagination.TenantInfo
	moveID     pulid.ID
	origin     *shipment.Stop
	middle     *shipment.Stop
	final      *shipment.Stop
	recorded   []*repositories.RecordStopActualRequest
}

func newStopActuals214Fixture(t *testing.T) *stopActuals214Fixture {
	t.Helper()

	commentRepo := mocks.NewMockShipmentCommentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveSvc := mocks.NewMockShipmentMoveService(t)

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
		moveSvc:  moveSvc,
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
		shipmentMoves:       moveSvc,
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
			func(req *repositories.GetMoveByIDRequest) bool {
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
	f.moveSvc.EXPECT().
		RecordStopActual(mock.Anything, mock.AnythingOfType("*repositories.RecordStopActualRequest")).
		RunAndReturn(
			func(_ context.Context, req *repositories.RecordStopActualRequest) (*shipment.ShipmentMove, error) {
				f.recorded = append(f.recorded, req)
				return f.move(), nil
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
	require.Equal(t, repositories.StopActualActionArrive, recorded.Action)
	require.Equal(t, fixture.origin.ID, recorded.StopID)
	require.Equal(t, fixture.moveID, recorded.MoveID)
	require.Equal(t, fixture.tenantInfo, recorded.TenantInfo)
	require.NotNil(t, recorded.OccurredAt)
	require.EqualValues(t, 1767787200, *recorded.OccurredAt)
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
	require.Equal(t, repositories.StopActualActionDepart, recorded.Action)
	require.Equal(t, fixture.origin.ID, recorded.StopID)
	require.Nil(t, recorded.OccurredAt)
}

func TestRecordTenderedShipmentStatus_X1ArrivesAtLastDestinationStop(t *testing.T) {
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
	require.Equal(t, repositories.StopActualActionArrive, fixture.recorded[0].Action)
	require.Equal(t, fixture.final.ID, fixture.recorded[0].StopID)
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
	require.Equal(t, repositories.StopActualActionArrive, fixture.recorded[0].Action)
	require.Equal(t, repositories.StopActualActionDepart, fixture.recorded[1].Action)
	require.Equal(t, fixture.final.ID, fixture.recorded[0].StopID)
	require.Equal(t, fixture.final.ID, fixture.recorded[1].StopID)
	require.NotNil(t, fixture.recorded[1].OccurredAt)
	require.EqualValues(t, 1767787200, *fixture.recorded[1].OccurredAt)
}

func TestRecordTenderedShipmentStatus_D1OnlyDepartsWhenAlreadyArrived(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	arrivedAt := int64(1767780000)
	fixture.final.ActualArrival = &arrivedAt
	fixture.final.Status = shipment.StopStatusInTransit
	fixture.expectMoveLoad()
	fixture.expectRecordStopActual(1)

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("D1", 1767787200),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorded, 1)
	require.Equal(t, repositories.StopActualActionDepart, fixture.recorded[0].Action)
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
	fixture.moveSvc.AssertNotCalled(t, "RecordStopActual", mock.Anything, mock.Anything)
}

func TestRecordTenderedShipmentStatus_StopActualBusinessErrorDegradesToWarning(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.moveSvc.EXPECT().
		RecordStopActual(mock.Anything, mock.AnythingOfType("*repositories.RecordStopActualRequest")).
		Return(nil, errortypes.NewBusinessError("Complete the earlier stops on this load first")).
		Once()

	warnings, err := fixture.service.RecordTenderedShipmentStatus(
		t.Context(),
		fixture.request("X1", 1767787200),
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "X1")
	require.Contains(t, warnings[0], "Complete the earlier stops on this load first")
}

func TestRecordTenderedShipmentStatus_StopActualInfrastructureErrorPropagates(t *testing.T) {
	t.Parallel()

	fixture := newStopActuals214Fixture(t)
	fixture.expectMoveLoad()
	fixture.moveSvc.EXPECT().
		RecordStopActual(mock.Anything, mock.AnythingOfType("*repositories.RecordStopActualRequest")).
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
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetMoveByIDRequest")).
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

	commentRepo := mocks.NewMockShipmentCommentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
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
		action          repositories.StopActualAction
		arriveIfMissing bool
	}{
		{
			code:   "X3",
			mapped: true,
			target: stopTargetOrigin,
			action: repositories.StopActualActionArrive,
		},
		{
			code:   "AF",
			mapped: true,
			target: stopTargetOrigin,
			action: repositories.StopActualActionDepart,
		},
		{
			code:   "X1",
			mapped: true,
			target: stopTargetDestination,
			action: repositories.StopActualActionArrive,
		},
		{
			code:            "D1",
			mapped:          true,
			target:          stopTargetDestination,
			action:          repositories.StopActualActionDepart,
			arriveIfMissing: true,
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

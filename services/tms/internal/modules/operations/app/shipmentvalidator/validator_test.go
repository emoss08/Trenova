package shipmentvalidator

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateCreate_RejectsNestedIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = pulid.MustNew("sm_")

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].id")
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].id")
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].shipmentMoveId")
}

func TestValidateCreate_DefaultsMissingEntryMethod(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.EntryMethodManual, entity.EntryMethod)
}

func TestValidateUpdate_RejectsDuplicateMoveIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1

	moveID := pulid.MustNew("sm_")
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].ID = moveID
	entity.Moves[1].ID = moveID

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].id")
}

func TestValidateUpdate_RejectsDuplicateStopIDsAcrossShipment(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[1].ID = pulid.MustNew("sm_")

	stopID := pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ID = stopID
	entity.Moves[1].Stops[0].ID = stopID

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[0].id")
}

func TestValidateUpdate_RejectsDuplicateStopSequencesWithinMove(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].Stops = []*shipment.Stop{
		shipmentservicetest.ValidStop(),
		shipmentservicetest.ValidStop(),
	}
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].Sequence = 0
	entity.Moves[0].Stops[1].Sequence = 0

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[1].sequence")
}

func TestValidateUpdate_RejectsDuplicateMoveSequences(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[1].ID = pulid.MustNew("sm_")
	entity.Moves[0].Sequence = 0
	entity.Moves[1].Sequence = 0

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].sequence")
}

func TestValidateUpdate_AllowsUniqueNestedIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[1].ID = pulid.MustNew("sm_")
	entity.Moves[0].Sequence = 0
	entity.Moves[1].Sequence = 1
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	entity.Moves[1].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[1].Stops[1].ID = pulid.MustNew("stp_")
	entity.Moves[1].Stops[0].Sequence = 0
	entity.Moves[1].Stops[1].Sequence = 1

	multiErr := v.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, multiErr)
}

func TestValidateCreate_RejectsAdditionalChargeNestedIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			ID:                  pulid.MustNew("ac_"),
			ShipmentID:          pulid.MustNew("shp_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              "Flat",
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[0].id")
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[0].shipmentId")
}

func TestValidateUpdate_RejectsDuplicateAdditionalChargeIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	chargeID := pulid.MustNew("ac_")
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			ID:                  chargeID,
			ShipmentID:          entity.ID,
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              "Flat",
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
		{
			ID:                  chargeID,
			ShipmentID:          entity.ID,
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              "Flat",
			Amount:              decimal.NewFromInt(20),
			Unit:                1,
		},
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[1].id")
}

func TestValidateCreate_RejectsShipmentCommodityNestedIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			ID:          pulid.MustNew("sc_"),
			ShipmentID:  pulid.MustNew("shp_"),
			CommodityID: pulid.MustNew("com_"),
			Weight:      100,
			Pieces:      10,
		},
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[0].id")
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[0].shipmentId")
}

func TestValidateUpdate_RejectsDuplicateShipmentCommodityIDs(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	commodityID := pulid.MustNew("sc_")
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			ID:          commodityID,
			ShipmentID:  entity.ID,
			CommodityID: pulid.MustNew("com_"),
			Weight:      100,
			Pieces:      10,
		},
		{
			ID:          commodityID,
			ShipmentID:  entity.ID,
			CommodityID: pulid.MustNew("com_"),
			Weight:      120,
			Pieces:      12,
		},
	}

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[1].id")
}

func TestValidateCreate_MapsShipmentCommodityFieldErrorsToNestedPath(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: pulid.MustNew("com_"),
			Weight:      0,
			Pieces:      1,
		},
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[0].weight")
	assertNoErrorField(t, multiErr, "weight")
}

func TestValidateCreate_MapsAdditionalChargeFieldErrorsToNestedPath(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              "Flat",
			Amount:              decimal.Zero,
			Unit:                1,
		},
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[0].amount")
	assertNoErrorField(t, multiErr, "amount")
}

func TestValidateCreate_RejectsActualDepartureBeforeActualArrival(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	actualArrival := int64(200)
	actualDeparture := int64(100)
	entity.Moves[0].Stops[0].ActualArrival = &actualArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualDeparture

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualDeparture")
}

func TestValidateCreate_RejectsActualArrivalInFuture(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	actualArrival := timeutils.NowUnix() + 3600
	entity.Moves[0].Stops[0].ActualArrival = &actualArrival

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualArrival")
}

func TestValidateUpdate_RejectsActualDepartureInFuture(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = entity.Moves[0].ID
	actualDeparture := timeutils.NowUnix() + 3600
	entity.Moves[0].Stops[0].ActualDeparture = &actualDeparture

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualDeparture")
}

func TestValidateUpdate_RejectsScheduledWindowEndBeforeArrivalOnNestedStop(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[1].ID = pulid.MustNew("sm_")
	entity.Moves[1].Stops[1].ID = pulid.MustNew("stp_")
	entity.Moves[1].Stops[1].ShipmentMoveID = entity.Moves[1].ID
	entity.Moves[1].Stops[1].ScheduledWindowStart = 200
	entity.Moves[1].Stops[1].ScheduledWindowEnd = new(int64(100))

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[1].scheduledWindowEnd")
}

func TestValidateCreate_RejectsMoveWithFewerThanTwoStops(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops = []*shipment.Stop{shipmentservicetest.ValidStop()}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops")
}

func TestValidateCreate_RejectsMoveWithNonPickupFirstStop(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops[0].Type = shipmentvocab.StopTypeDelivery

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].type")
}

func TestValidateCreate_RejectsMoveWithNonDeliveryLastStop(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops[1].Type = shipmentvocab.StopTypePickup

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[1].type")
}

func TestValidateCreate_RejectsDeliveryBeforePickup(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops = []*shipment.Stop{
		{
			LocationID:           pulid.MustNew("loc_"),
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopTypeDelivery,
			ScheduleType:         shipmentvocab.StopScheduleTypeOpen,
			Sequence:             0,
			ScheduledWindowStart: 1,
			ScheduledWindowEnd:   new(int64(2)),
		},
		{
			LocationID:           pulid.MustNew("loc_"),
			Status:               shipmentvocab.StopStatusNew,
			Type:                 shipmentvocab.StopTypeDelivery,
			ScheduleType:         shipmentvocab.StopScheduleTypeOpen,
			Sequence:             1,
			ScheduledWindowStart: 3,
			ScheduledWindowEnd:   new(int64(4)),
		},
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].type")
}

func TestValidateCreate_RejectsMoveWithOutOfOrderPlannedStops(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops[0].ScheduledWindowEnd = new(int64(10))
	entity.Moves[0].Stops[1].ScheduledWindowStart = 5

	multiErr := v.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].scheduledWindowEnd")
}

func TestValidateUpdate_RejectsMoveWithOutOfOrderActualStops(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = entity.Moves[0].ID
	entity.Moves[0].Stops[1].ShipmentMoveID = entity.Moves[0].ID
	actualDeparture := int64(20)
	nextArrival := int64(15)
	entity.Moves[0].Stops[0].ActualDeparture = &actualDeparture
	entity.Moves[0].Stops[1].ActualArrival = &nextArrival

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualDeparture")
}

func TestValidateCreate_AllowsLegacyCompatibleMixedStopSequence(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops = []*shipment.Stop{
		shipmentservicetest.MakeStop(shipmentvocab.StopTypePickup, 0, 1, 2),
		shipmentservicetest.MakeStop(shipmentvocab.StopTypeDelivery, 1, 3, 4),
		shipmentservicetest.MakeStop(shipmentvocab.StopTypeSplitPickup, 2, 5, 6),
		shipmentservicetest.MakeStop(shipmentvocab.StopTypeSplitDelivery, 3, 7, 8),
	}

	multiErr := v.ValidateCreate(t.Context(), entity)

	assert.Nil(t, multiErr)
}

func TestValidateUpdate_RejectsActualArrivalBeforePreviousTractorEvent(t *testing.T) {
	t.Parallel()

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	allowNoExternalTimelineOverlap(assignmentRepo)
	v := newTestValidatorWithAssignmentRepo(t, assignmentRepo)

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].ShipmentID = entity.ID
	entity.Moves[0].Assignment = validAssignmentForValidation(
		entity.Moves[0],
		entity.OrganizationID,
		entity.BusinessUnitID,
	)
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = entity.Moves[0].ID
	actualArrival := int64(100)
	entity.Moves[0].Stops[0].ActualArrival = &actualArrival

	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.MatchedBy(func(req assignmentports.FindNearestActualTimelineEventRequest) bool {
				return req.TenantInfo.OrgID == entity.OrganizationID &&
					req.TenantInfo.BuID == entity.BusinessUnitID &&
					req.ExcludeShipmentID == entity.ID &&
					req.Timestamp == actualArrival &&
					req.Direction == assignmentports.ActualTimelineDirectionPrevious
			}),
			*entity.Moves[0].Assignment.TractorID,
		).
		Return(&assignmentports.ActualTimelineEvent{
			Timestamp: 100,
			EventType: assignmentports.ActualTimelineEventTypeDeparture,
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.MatchedBy(func(req assignmentports.FindNearestActualTimelineEventRequest) bool {
				return req.Direction == assignmentports.ActualTimelineDirectionNext &&
					req.Timestamp == actualArrival
			}),
			*entity.Moves[0].Assignment.TractorID,
		).
		Return(nil, nil).
		Once()
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			*entity.Moves[0].Assignment.PrimaryWorkerID,
		).
		Return(nil, nil).
		Twice()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualArrival")
}

func TestValidateUpdate_RejectsActualDepartureAfterNextPrimaryWorkerEvent(t *testing.T) {
	t.Parallel()

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	allowNoExternalTimelineOverlap(assignmentRepo)
	v := newTestValidatorWithAssignmentRepo(t, assignmentRepo)

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].ShipmentID = entity.ID
	entity.Moves[0].Assignment = validAssignmentForValidation(
		entity.Moves[0],
		entity.OrganizationID,
		entity.BusinessUnitID,
	)
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = entity.Moves[0].ID
	actualDeparture := int64(200)
	entity.Moves[0].Stops[0].ActualDeparture = &actualDeparture

	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.MatchedBy(func(req assignmentports.FindNearestActualTimelineEventRequest) bool {
				return req.TenantInfo.OrgID == entity.OrganizationID &&
					req.TenantInfo.BuID == entity.BusinessUnitID &&
					req.ExcludeShipmentID == entity.ID &&
					req.Timestamp == actualDeparture &&
					req.Direction == assignmentports.ActualTimelineDirectionPrevious
			}),
			*entity.Moves[0].Assignment.PrimaryWorkerID,
		).
		Return(nil, nil).
		Once()
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.MatchedBy(func(req assignmentports.FindNearestActualTimelineEventRequest) bool {
				return req.Direction == assignmentports.ActualTimelineDirectionNext &&
					req.Timestamp == actualDeparture
			}),
			*entity.Moves[0].Assignment.PrimaryWorkerID,
		).
		Return(&assignmentports.ActualTimelineEvent{
			Timestamp: 200,
			EventType: assignmentports.ActualTimelineEventTypeArrival,
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			*entity.Moves[0].Assignment.TractorID,
		).
		Return(nil, nil).
		Twice()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualDeparture")
}

func TestValidateUpdate_RejectsOverlappingActualTimesWithinShipmentForSameTractor(t *testing.T) {
	t.Parallel()

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	allowNoExternalTimelineOverlap(assignmentRepo)
	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil)
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil)
	v := newTestValidatorWithAssignmentRepo(t, assignmentRepo)
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}

	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")
	for idx, move := range entity.Moves {
		move.ID = pulid.MustNew("sm_")
		move.ShipmentID = entity.ID
		move.Assignment = &shipment.Assignment{
			ID:              pulid.MustNew("asn_"),
			ShipmentMoveID:  move.ID,
			OrganizationID:  entity.OrganizationID,
			BusinessUnitID:  entity.BusinessUnitID,
			PrimaryWorkerID: &workerID,
			TractorID:       &tractorID,
			Status:          shipmentvocab.AssignmentStatusNew,
		}
		move.Stops[0].ID = pulid.MustNew("stp_")
		move.Stops[0].ShipmentMoveID = move.ID
		move.Stops[1].ID = pulid.MustNew("stp_")
		move.Stops[1].ShipmentMoveID = move.ID
		move.Sequence = int64(idx)
	}

	firstArrival := int64(100)
	firstDeparture := int64(110)
	secondArrival := int64(105)
	secondDeparture := int64(115)

	entity.Moves[0].Stops[0].ActualArrival = &firstArrival
	entity.Moves[0].Stops[0].ActualDeparture = &firstDeparture
	entity.Moves[1].Stops[0].ActualArrival = &secondArrival
	entity.Moves[1].Stops[0].ActualDeparture = &secondDeparture

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[1].stops[0].actualArrival")
}

func TestValidateUpdate_RejectsActualArrivalWithinPersistedTractorWindow(t *testing.T) {
	t.Parallel()

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	v := newTestValidatorWithAssignmentRepo(t, assignmentRepo)

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Moves[0].ID = pulid.MustNew("sm_")
	entity.Moves[0].ShipmentID = entity.ID
	entity.Moves[0].Assignment = validAssignmentForValidation(
		entity.Moves[0],
		entity.OrganizationID,
		entity.BusinessUnitID,
	)
	entity.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	entity.Moves[0].Stops[0].ShipmentMoveID = entity.Moves[0].ID

	actualArrival := int64(1773837000)
	entity.Moves[0].Stops[0].ActualArrival = &actualArrival

	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			*entity.Moves[0].Assignment.TractorID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			*entity.Moves[0].Assignment.PrimaryWorkerID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.MatchedBy(func(req assignmentports.FindOverlappingActualTimelineWindowRequest) bool {
				return req.TenantInfo.OrgID == entity.OrganizationID &&
					req.TenantInfo.BuID == entity.BusinessUnitID &&
					req.ExcludeShipmentID == entity.ID &&
					req.Timestamp == actualArrival
			}),
			*entity.Moves[0].Assignment.TractorID,
		).
		Return(&assignmentports.ActualTimelineWindow{
			StartTimestamp: 1773835200,
			EndTimestamp:   1773837120,
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			*entity.Moves[0].Assignment.PrimaryWorkerID,
		).
		Return(nil, nil).
		Once()

	multiErr := v.ValidateUpdate(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualArrival")
}

func TestValidateUpdateWithOriginal_SkipsExternalTimelineLookupForUnchangedActuals(t *testing.T) {
	t.Parallel()

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	v := newTestValidatorWithAssignmentRepo(t, assignmentRepo)

	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].ShipmentID = original.ID
	original.Moves[0].Assignment = validAssignmentForValidation(
		original.Moves[0],
		original.OrganizationID,
		original.BusinessUnitID,
	)
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	arrival := int64(100)
	departure := int64(110)
	original.Moves[0].Stops[0].ActualArrival = &arrival
	original.Moves[0].Stops[0].ActualDeparture = &departure

	updated := shipmentservicetest.CloneShipment(original)

	multiErr := v.ValidateUpdateWithOriginal(t.Context(), original, updated)

	assert.Nil(t, multiErr)
}

func TestValidateUpdateWithOriginal_PreservesOriginalEntryMethod(t *testing.T) {
	t.Parallel()

	v := newTestValidator(t)
	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.EntryMethod = shipmentvocab.EntryMethodEDI

	updated := shipmentservicetest.CloneShipment(original)
	updated.EntryMethod = ""

	multiErr := v.ValidateUpdateWithOriginal(t.Context(), original, updated)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.EntryMethodEDI, updated.EntryMethod)
}

func validAssignmentForValidation(
	move *shipment.ShipmentMove,
	orgID pulid.ID,
	buID pulid.ID,
) *shipment.Assignment {
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trc_")

	return &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		ShipmentMoveID:  move.ID,
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		PrimaryWorkerID: &workerID,
		TractorID:       &tractorID,
		Status:          shipmentvocab.AssignmentStatusNew,
	}
}

func assertNoErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	for _, validationErr := range multiErr.Errors {
		if validationErr.Field == field {
			t.Fatalf("unexpected validation error on field %s", field)
		}
	}
}

func allowNoExternalTimelineOverlap(
	assignmentRepo *assignmentportsmock.MockAssignmentRepository,
) {
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil)
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			mock.AnythingOfType("pulid.ID"),
		).
		Return(nil, nil)
}

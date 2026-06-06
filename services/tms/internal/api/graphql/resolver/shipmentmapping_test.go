package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentFromInput_MapsNestedPayloads(t *testing.T) {
	t.Parallel()

	authCtx := &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	shipmentID := pulid.MustNew("shp_")
	serviceTypeID := pulid.MustNew("svc_")
	shipmentTypeID := pulid.MustNew("sht_")
	customerID := pulid.MustNew("cus_")
	formulaTemplateID := pulid.MustNew("ft_")
	moveID := pulid.MustNew("mov_")
	stopID := pulid.MustNew("stp_")
	locationID := pulid.MustNew("loc_")
	chargeID := pulid.MustNew("chg_")
	accessorialChargeID := pulid.MustNew("acc_")
	commodityID := pulid.MustNew("cmd_")
	status := gqlmodel.ShipmentStatusInTransit
	moveStatus := gqlmodel.MoveStatusAssigned
	stopStatus := gqlmodel.StopStatusInTransit
	stopType := gqlmodel.StopTypeDelivery
	scheduleType := gqlmodel.StopScheduleTypeAppointment
	loaded := false
	sequence := 2
	stopSequence := 3
	scheduledStart := 1_800_000_000
	amount := "12.34"
	unit := 4
	pieces := 6
	weight := 700
	ratingUnit := 2

	entity, err := shipmentFromInput(gqlmodel.ShipmentInput{
		ServiceTypeID:         serviceTypeID.String(),
		ShipmentTypeID:        shipmentTypeID.String(),
		CustomerID:            customerID.String(),
		FormulaTemplateID:     formulaTemplateID.String(),
		Status:                &status,
		ProNumber:             testStringPtr("SHP-100"),
		OtherChargeAmount:     testStringPtr("1.25"),
		FreightChargeAmount:   testStringPtr("2.50"),
		BaseRate:              testStringPtr("3.75"),
		TotalChargeAmount:     testStringPtr("5.00"),
		TemperatureMin:        testIntPtr(-10),
		TemperatureMax:        testIntPtr(42),
		RatingUnit:            &ratingUnit,
		SourceDocumentID:      testStringPtr("doc_123"),
		BillingTransferStatus: testStringPtr("Pending"),
		Moves: []*gqlmodel.ShipmentMoveInput{
			{
				ID:       testStringPtr(moveID.String()),
				Status:   &moveStatus,
				Loaded:   &loaded,
				Sequence: &sequence,
				Stops: []*gqlmodel.ShipmentStopInput{
					{
						ID:                   testStringPtr(stopID.String()),
						LocationID:           locationID.String(),
						Status:               &stopStatus,
						Type:                 &stopType,
						ScheduleType:         &scheduleType,
						Sequence:             &stopSequence,
						ScheduledWindowStart: &scheduledStart,
					},
				},
			},
		},
		AdditionalCharges: []*gqlmodel.ShipmentAdditionalChargeInput{
			{
				ID:                  testStringPtr(chargeID.String()),
				AccessorialChargeID: accessorialChargeID.String(),
				Amount:              &amount,
				Unit:                &unit,
			},
		},
		Commodities: []*gqlmodel.ShipmentCommodityInput{
			{
				CommodityID: commodityID.String(),
				Pieces:      &pieces,
				Weight:      &weight,
			},
		},
	}, shipmentID, authCtx)
	require.NoError(t, err)

	assert.Equal(t, shipmentID, entity.ID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, shipmentdomain.StatusInTransit, entity.Status)
	assert.Equal(t, "SHP-100", entity.ProNumber)
	assert.Equal(t, "1.25", entity.OtherChargeAmount.Decimal.String())
	assert.Equal(t, "2.5", entity.FreightChargeAmount.Decimal.String())
	assert.Equal(t, "3.75", entity.BaseRate.Decimal.String())
	assert.Equal(t, "5", entity.TotalChargeAmount.Decimal.String())
	require.NotNil(t, entity.TemperatureMin)
	require.NotNil(t, entity.TemperatureMax)
	assert.Equal(t, int16(-10), *entity.TemperatureMin)
	assert.Equal(t, int16(42), *entity.TemperatureMax)
	assert.Equal(t, int64(2), entity.RatingUnit)
	assert.Equal(t, "doc_123", entity.SourceDocumentID)

	require.Len(t, entity.Moves, 1)
	assert.Equal(t, moveID, entity.Moves[0].ID)
	assert.Equal(t, shipmentdomain.MoveStatusAssigned, entity.Moves[0].Status)
	assert.False(t, entity.Moves[0].Loaded)
	assert.Equal(t, int64(2), entity.Moves[0].Sequence)

	require.Len(t, entity.Moves[0].Stops, 1)
	assert.Equal(t, stopID, entity.Moves[0].Stops[0].ID)
	assert.Equal(t, locationID, entity.Moves[0].Stops[0].LocationID)
	assert.Equal(t, shipmentdomain.StopStatusInTransit, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipmentdomain.StopTypeDelivery, entity.Moves[0].Stops[0].Type)
	assert.Equal(t, shipmentdomain.StopScheduleTypeAppointment, entity.Moves[0].Stops[0].ScheduleType)
	assert.Equal(t, int64(3), entity.Moves[0].Stops[0].Sequence)
	assert.Equal(t, int64(1_800_000_000), entity.Moves[0].Stops[0].ScheduledWindowStart)

	require.Len(t, entity.AdditionalCharges, 1)
	assert.Equal(t, chargeID, entity.AdditionalCharges[0].ID)
	assert.Equal(t, accessorialChargeID, entity.AdditionalCharges[0].AccessorialChargeID)
	assert.Equal(t, "12.34", entity.AdditionalCharges[0].Amount.String())
	assert.Equal(t, int16(4), entity.AdditionalCharges[0].Unit)

	require.Len(t, entity.Commodities, 1)
	assert.Equal(t, commodityID, entity.Commodities[0].CommodityID)
	assert.Equal(t, int64(6), entity.Commodities[0].Pieces)
	assert.Equal(t, int64(700), entity.Commodities[0].Weight)
}

func TestShipmentFromInput_RejectsOutOfRangeTemperature(t *testing.T) {
	t.Parallel()

	_, err := shipmentFromInput(gqlmodel.ShipmentInput{
		ServiceTypeID:     pulid.MustNew("svc_").String(),
		ShipmentTypeID:    pulid.MustNew("sht_").String(),
		CustomerID:        pulid.MustNew("cus_").String(),
		FormulaTemplateID: pulid.MustNew("ft_").String(),
		TemperatureMin:    testIntPtr(40_000),
	}, pulid.Nil, &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "temperatureMin")
}

func TestShipmentEventToModel_MapsOptionalFieldsAndRelations(t *testing.T) {
	t.Parallel()

	eventID := pulid.MustNew("se_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")
	stopID := pulid.MustNew("stp_")
	actorID := pulid.MustNew("usr_")
	actor := &tenant.User{
		ID:           actorID,
		Name:         "Maria Diaz",
		EmailAddress: "maria@example.com",
	}
	event := &shipmentevent.Event{
		ID:             eventID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     shipmentID,
		StopID:         stopID,
		Type:           shipmentevent.TypeStopCompleted,
		Severity:       shipmentevent.SeveritySuccess,
		ActorType:      shipmentevent.ActorUser,
		ActorID:        actorID,
		ActorLabel:     "Maria Diaz",
		Summary:        "Stop completed",
		Metadata:       map[string]any{"stopSequence": float64(2)},
		OccurredAt:     1_800_000_000,
		CorrelationID:  "corr-1",
		Actor:          actor,
		Shipment: &shipmentdomain.Shipment{
			ID:        shipmentID,
			ProNumber: "SHP-100",
		},
	}

	model, err := shipmentEventToModel(event)
	require.NoError(t, err)

	assert.Equal(t, eventID.String(), model.ID)
	assert.Equal(t, orgID.String(), model.OrganizationID)
	assert.Equal(t, buID.String(), model.BusinessUnitID)
	assert.Equal(t, shipmentID.String(), model.ShipmentID)
	assert.Nil(t, model.MoveID)
	assert.Equal(t, stopID.String(), *model.StopID)
	assert.Nil(t, model.AssignmentID)
	assert.Nil(t, model.CommentID)
	assert.Nil(t, model.HoldID)
	assert.Equal(t, gqlmodel.ShipmentEventTypeStopCompleted, model.Type)
	assert.Equal(t, gqlmodel.ShipmentEventSeveritySuccess, model.Severity)
	assert.Equal(t, gqlmodel.ShipmentEventActorTypeUser, model.ActorType)
	assert.Equal(t, actorID.String(), *model.ActorID)
	assert.Equal(t, "Maria Diaz", model.ActorLabel)
	assert.Equal(t, "Stop completed", model.Summary)
	assert.Equal(t, map[string]any{"stopSequence": float64(2)}, model.Metadata)
	assert.Equal(t, 1_800_000_000, model.OccurredAt)
	assert.Equal(t, "corr-1", *model.CorrelationID)
	assert.Same(t, actor, model.Actor)
	require.NotNil(t, model.Shipment)
	assert.Equal(t, shipmentID.String(), *model.Shipment.ID)
	assert.Equal(t, "SHP-100", *model.Shipment.ProNumber)
}

func testIntPtr(value int) *int {
	return &value
}

func testStringPtr(value string) *string {
	return &value
}

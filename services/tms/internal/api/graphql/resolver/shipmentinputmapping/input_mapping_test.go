package shipmentinputmapping

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
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

	entity, err := ShipmentFromInput(&gqlmodel.ShipmentInput{
		ServiceTypeID:         serviceTypeID.String(),
		ShipmentTypeID:        shipmentTypeID.String(),
		CustomerID:            customerID.String(),
		FormulaTemplateID:     new(formulaTemplateID.String()),
		Status:                &status,
		ProNumber:             new("SHP-100"),
		OtherChargeAmount:     new("1.25"),
		FreightChargeAmount:   new("2.50"),
		TariffInputs:          map[string]any{"rate": map[string]any{"$q": "3.75", "u": "$/mi"}},
		TotalChargeAmount:     new("5.00"),
		TemperatureMin:        new(-10),
		TemperatureMax:        new(42),
		SourceDocumentID:      new("doc_123"),
		BillingTransferStatus: new("Pending"),
		Moves: []*gqlmodel.ShipmentMoveInput{
			{
				ID:       new(moveID.String()),
				Status:   &moveStatus,
				Loaded:   &loaded,
				Sequence: &sequence,
				Stops: []*gqlmodel.ShipmentStopInput{
					{
						ID:                   new(stopID.String()),
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
				ID:                  new(chargeID.String()),
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
	assert.Equal(t, shipmentvocab.StatusInTransit, entity.Status)
	assert.Equal(t, "SHP-100", entity.ProNumber)
	assert.Equal(t, "1.25", entity.OtherChargeAmount.Decimal.String())
	assert.Equal(t, "2.5", entity.FreightChargeAmount.Decimal.String())
	rate, err := rules.DecodeQuantity(entity.TariffInputs["rate"])
	require.NoError(t, err)
	assert.Equal(t, "3.75", rate.Value.String())
	assert.Equal(t, "$/mi", rate.Unit)
	assert.Equal(t, "5", entity.TotalChargeAmount.Decimal.String())
	require.NotNil(t, entity.TemperatureMin)
	require.NotNil(t, entity.TemperatureMax)
	assert.Equal(t, int16(-10), *entity.TemperatureMin)
	assert.Equal(t, int16(42), *entity.TemperatureMax)
	assert.Equal(t, "doc_123", entity.SourceDocumentID)

	require.Len(t, entity.Moves, 1)
	assert.Equal(t, moveID, entity.Moves[0].ID)
	assert.Equal(t, shipmentvocab.MoveStatusAssigned, entity.Moves[0].Status)
	assert.False(t, entity.Moves[0].Loaded)
	assert.Equal(t, int64(2), entity.Moves[0].Sequence)

	require.Len(t, entity.Moves[0].Stops, 1)
	assert.Equal(t, stopID, entity.Moves[0].Stops[0].ID)
	assert.Equal(t, locationID, entity.Moves[0].Stops[0].LocationID)
	assert.Equal(t, shipmentvocab.StopStatusInTransit, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipmentvocab.StopTypeDelivery, entity.Moves[0].Stops[0].Type)
	assert.Equal(
		t,
		shipmentvocab.StopScheduleTypeAppointment,
		entity.Moves[0].Stops[0].ScheduleType,
	)
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

	_, err := ShipmentFromInput(&gqlmodel.ShipmentInput{
		ServiceTypeID:     pulid.MustNew("svc_").String(),
		ShipmentTypeID:    pulid.MustNew("sht_").String(),
		CustomerID:        pulid.MustNew("cus_").String(),
		FormulaTemplateID: new(pulid.MustNew("ft_").String()),
		TemperatureMin:    new(40_000),
	}, pulid.Nil, &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "temperatureMin")
}

func TestShipmentFromInput_AllowsAShipmentRatedByHand(t *testing.T) {
	t.Parallel()

	for _, id := range []pulid.ID{pulid.Nil, pulid.MustNew("shp_")} {
		entity, err := ShipmentFromInput(&gqlmodel.ShipmentInput{
			ServiceTypeID:  pulid.MustNew("svc_").String(),
			ShipmentTypeID: pulid.MustNew("sht_").String(),
			CustomerID:     pulid.MustNew("cus_").String(),
		}, id, &authctx.AuthContext{
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
		})

		require.NoError(t, err)
		assert.True(t, entity.FormulaTemplateID.IsNil())
		assert.True(t, entity.RatedByHand())
		assert.NotNil(t, entity.TariffInputs)
	}
}

// A value that is present but not an identifier is a client bug either way, and
// the field has to be named or the form has nothing to attach the error to.
func TestShipmentFromInput_RejectsAMalformedRatingMethodOnANewShipment(t *testing.T) {
	t.Parallel()

	_, err := ShipmentFromInput(&gqlmodel.ShipmentInput{
		ServiceTypeID:     pulid.MustNew("svc_").String(),
		ShipmentTypeID:    pulid.MustNew("sht_").String(),
		CustomerID:        pulid.MustNew("cus_").String(),
		FormulaTemplateID: new("not-an-id"),
	}, pulid.Nil, &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	})

	require.Error(t, err)
	assertRatingMethodError(t, err, errortypes.ErrInvalid)
}

// The message a validation error carries names the problem, not the control it
// belongs to, so the field has to be read off the error itself — it is what the
// form attaches the message to.
func assertRatingMethodError(t *testing.T, err error, code errortypes.ErrorCode) {
	t.Helper()

	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "formulaTemplateId", typed.Field)
	assert.Equal(t, code, typed.Code)
}

func TestShipmentCommodityMapping_CarriesDimensionsBothWays(t *testing.T) {
	t.Parallel()

	authCtx := &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}
	commodityID := pulid.MustNew("com_")
	length, width, height := 48.5, 8.5, 13.25

	entities, err := shipmentCommoditiesFromInput([]*gqlmodel.ShipmentCommodityInput{
		{
			CommodityID: commodityID.String(),
			Pieces:      ptrTo(4),
			Weight:      ptrTo(1200),
			LengthFeet:  &length,
			WidthFeet:   &width,
			HeightFeet:  &height,
		},
	}, authCtx)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	require.NotNil(t, entities[0].LengthFeet)
	require.NotNil(t, entities[0].WidthFeet)
	require.NotNil(t, entities[0].HeightFeet)
	assert.InDelta(t, length, *entities[0].LengthFeet, 0.001)
	assert.InDelta(t, width, *entities[0].WidthFeet, 0.001)
	assert.InDelta(t, height, *entities[0].HeightFeet, 0.001)

	models, err := shipmentmapping.ShipmentCommoditiesToModel(entities)

	require.NoError(t, err)
	require.Len(t, models, 1)
	require.NotNil(t, models[0].LengthFeet)
	require.NotNil(t, models[0].WidthFeet)
	require.NotNil(t, models[0].HeightFeet)
	assert.InDelta(t, length, *models[0].LengthFeet, 0.001)
	assert.InDelta(t, width, *models[0].WidthFeet, 0.001)
	assert.InDelta(t, height, *models[0].HeightFeet, 0.001)
}

func TestShipmentCommodityMapping_LeavesDimensionsNilWhenAbsent(t *testing.T) {
	t.Parallel()

	authCtx := &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}

	entities, err := shipmentCommoditiesFromInput([]*gqlmodel.ShipmentCommodityInput{
		{CommodityID: pulid.MustNew("com_").String(), Pieces: ptrTo(1), Weight: ptrTo(10)},
	}, authCtx)

	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Nil(t, entities[0].LengthFeet)
	assert.Nil(t, entities[0].WidthFeet)
	assert.Nil(t, entities[0].HeightFeet)
}

func ptrTo[T any](v T) *T { return &v }

package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

func TestShipmentToModel_MapsTypedRelationFields(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	customerID := pulid.MustNew("cus_")
	formulaTemplateID := pulid.MustNew("ft_")
	accessorialChargeID := pulid.MustNew("acc_")
	commodityID := pulid.MustNew("cmd_")
	hazmatID := pulid.MustNew("hm_")
	sourceVersion := int64(2)
	entity := &shipmentdomain.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    buID,
		OrganizationID:    orgID,
		ServiceTypeID:     pulid.MustNew("svc_"),
		ShipmentTypeID:    pulid.MustNew("sht_"),
		CustomerID:        customerID,
		FormulaTemplateID: formulaTemplateID,
		Status:            shipmentdomain.StatusNew,
		EntryMethod:       shipmentdomain.EntryMethodManual,
		ProNumber:         "SHP-100",
		RatingUnit:        1,
		Customer: &customer.Customer{
			ID:                    customerID,
			BusinessUnitID:        buID,
			OrganizationID:        orgID,
			StateID:               pulid.MustNew("st_"),
			Status:                domaintypes.StatusActive,
			Code:                  "CUST",
			Name:                  "Acme",
			AddressLine1:          "1 Main",
			City:                  "Chicago",
			PostalCode:            "60601",
			ConsolidationPriority: 3,
			Version:               4,
			CreatedAt:             10,
			UpdatedAt:             11,
		},
		FormulaTemplate: &formulatemplate.FormulaTemplate{
			ID:                   formulaTemplateID,
			BusinessUnitID:       buID,
			OrganizationID:       orgID,
			Name:                 "Standard",
			Description:          "Standard tariff",
			Type:                 formulatemplate.TemplateTypeFreightCharge,
			Expression:           "base",
			Status:               formulatemplate.StatusActive,
			SchemaID:             "shipment",
			Metadata:             map[string]any{"mode": "auto"},
			Version:              5,
			SourceVersionNumber:  &sourceVersion,
			CurrentVersionNumber: 6,
			CreatedAt:            12,
			UpdatedAt:            13,
		},
		AdditionalCharges: []*shipmentdomain.AdditionalCharge{
			{
				ID:                  pulid.MustNew("sac_"),
				BusinessUnitID:      buID,
				OrganizationID:      orgID,
				ShipmentID:          pulid.MustNew("shp_"),
				AccessorialChargeID: accessorialChargeID,
				Amount:              decimal.RequireFromString("12.50"),
				Unit:                1,
				AccessorialCharge: &accessorialcharge.AccessorialCharge{
					ID:             accessorialChargeID,
					BusinessUnitID: buID,
					OrganizationID: orgID,
					Code:           "LFT",
					Description:    "Liftgate",
					Status:         domaintypes.StatusActive,
					Method:         accessorialcharge.MethodFlat,
					RateUnit:       accessorialcharge.RateUnitStop,
					Amount:         decimal.RequireFromString("10.00"),
					Version:        7,
					CreatedAt:      14,
					UpdatedAt:      15,
				},
			},
		},
		Commodities: []*shipmentdomain.ShipmentCommodity{
			{
				ID:             pulid.MustNew("shc_"),
				BusinessUnitID: buID,
				OrganizationID: orgID,
				ShipmentID:     pulid.MustNew("shp_"),
				CommodityID:    commodityID,
				Pieces:         2,
				Weight:         300,
				Commodity: &commodity.Commodity{
					ID:                  commodityID,
					BusinessUnitID:      buID,
					OrganizationID:      orgID,
					HazardousMaterialID: hazmatID,
					Status:              domaintypes.StatusActive,
					Name:                "Widgets",
					Description:         "Stacked widgets",
					FreightClass:        commodity.FreightClass100,
					LoadingInstructions: "Stack",
					Stackable:           true,
					Version:             8,
					CreatedAt:           16,
					UpdatedAt:           17,
				},
			},
		},
	}

	model, err := shipmentToModel(entity)
	require.NoError(t, err)

	require.NotNil(t, model.Customer)
	assert.Equal(t, customerID.String(), model.Customer.ID)
	assert.Equal(t, "Acme", model.Customer.Name)
	require.NotNil(t, model.FormulaTemplate)
	assert.Equal(t, "Standard", model.FormulaTemplate.Name)
	assert.Equal(t, map[string]any{"mode": "auto"}, model.FormulaTemplate.Metadata)
	require.Len(t, model.AdditionalCharges, 1)
	require.NotNil(t, model.AdditionalCharges[0].AccessorialCharge)
	assert.Equal(t, "LFT", model.AdditionalCharges[0].AccessorialCharge.Code)
	assert.Equal(t, "10", model.AdditionalCharges[0].AccessorialCharge.Amount)
	require.Len(t, model.Commodities, 1)
	require.NotNil(t, model.Commodities[0].Commodity)
	assert.Equal(t, "Widgets", model.Commodities[0].Commodity.Name)
	assert.Equal(t, hazmatID.String(), *model.Commodities[0].Commodity.HazardousMaterialID)
}

func TestShipmentBillingReadinessToModel_MapsWarningContext(t *testing.T) {
	t.Parallel()

	model := shipmentBillingReadinessToModel(&services.ShipmentBillingReadiness{
		ShipmentID:     "shp_1",
		ShipmentStatus: shipmentdomain.StatusCompleted,
		Policy: services.ShipmentBillingReadinessPolicy{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelWarn,
			RateValidationEnforcement:             tenant.EnforcementLevelWarn,
		},
		Warnings: []services.ShipmentBillingWarning{
			{
				Code:    "missing_documents",
				Message: "Missing documents",
				Context: map[string]any{
					"documentTypeId":          "dt_1",
					"documentTypeCode":        "POD",
					"documentTypeName":        "Proof of Delivery",
					"documentCount":           1,
					"requirementCount":        float64(2),
					"missingRequirementCount": int64(1),
					"serviceFailureIds":       []any{"sf_1", "sf_2"},
					"unresolvedCount":         2,
				},
			},
		},
	})

	require.Len(t, model.Warnings, 1)
	require.NotNil(t, model.Warnings[0].Context)
	assert.Equal(t, "dt_1", *model.Warnings[0].Context.DocumentTypeID)
	assert.Equal(t, "POD", *model.Warnings[0].Context.DocumentTypeCode)
	assert.Equal(t, "Proof of Delivery", *model.Warnings[0].Context.DocumentTypeName)
	assert.Equal(t, 1, *model.Warnings[0].Context.DocumentCount)
	assert.Equal(t, 2, *model.Warnings[0].Context.RequirementCount)
	assert.Equal(t, 1, *model.Warnings[0].Context.MissingRequirementCount)
	assert.Equal(t, []string{"sf_1", "sf_2"}, model.Warnings[0].Context.ServiceFailureIds)
	assert.Equal(t, 2, *model.Warnings[0].Context.UnresolvedCount)
}

func TestShipmentAnalyticsToModel_MapsTypedCards(t *testing.T) {
	t.Parallel()

	target := 96.5
	model, err := shipmentAnalyticsToModel(services.AnalyticsData{
		"savedViewCounts": map[string]any{
			"all":              10,
			"transit":          4,
			"at-risk":          2,
			"unassigned":       3,
			"delivering-today": 1,
		},
		"activeShipments": map[string]any{
			"count":               12,
			"changeFromYesterday": -1,
			"sparkline": []any{
				map[string]any{"hour": "08:00", "value": 3.5},
			},
			"breakdown": map[string]any{
				"inTransit": 6,
				"atRisk":    2,
				"loading":   3,
				"done":      1,
			},
		},
		"onTimePercent": map[string]any{
			"percent":         94.2,
			"onTimeCount":     94,
			"totalCount":      100,
			"target":          target,
			"deltaPp":         -1.1,
			"sevenDayPercent": 95.3,
		},
		"tomorrowsPickups": map[string]any{
			"date": "2026-06-07",
			"pickups": []any{
				map[string]any{"shipmentId": "shp_1", "proNumber": "PRO-1", "pickupWindowStart": 10, "customer": "Acme", "origin": "CHI", "destination": "DAL", "driver": "Alex", "status": "scheduled"},
			},
		},
		"laneHeatmap": map[string]any{
			"windowDays": 7,
			"total":      1,
			"cells": []any{
				map[string]any{"origin": "IL", "destination": "TX", "count": 1},
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, model.SavedViewCounts)
	assert.Equal(t, 10, *model.SavedViewCounts.All)
	require.NotNil(t, model.ActiveShipments)
	assert.Equal(t, 12, model.ActiveShipments.Count)
	require.Len(t, model.ActiveShipments.Sparkline, 1)
	assert.Equal(t, "08:00", model.ActiveShipments.Sparkline[0].Hour)
	require.NotNil(t, model.OnTimePercent)
	assert.Equal(t, target, *model.OnTimePercent.Target)
	require.NotNil(t, model.TomorrowsPickups)
	assert.Equal(t, "PRO-1", model.TomorrowsPickups.Pickups[0].ProNumber)
	require.NotNil(t, model.LaneHeatmap)
	assert.Equal(t, "TX", model.LaneHeatmap.Cells[0].Destination)
}

func testIntPtr(value int) *int {
	return &value
}

func testStringPtr(value string) *string {
	return &value
}

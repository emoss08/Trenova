package base

import (
	"testing"

	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			Source:               "tariff \"Standard\"\n",
			TestCount:            2,
			Status:               formulatemplate.StatusActive,
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

	model, err := ShipmentToModel(t.Context(), entity)
	require.NoError(t, err)

	require.NotNil(t, model.Customer)
	assert.Equal(t, customerID.String(), model.Customer.ID)
	assert.Equal(t, "Acme", model.Customer.Name)
	require.NotNil(t, model.FormulaTemplate)
	assert.Equal(t, "Standard", model.FormulaTemplate.Name)
	assert.Equal(t, "tariff \"Standard\"\n", model.FormulaTemplate.Source)
	assert.Equal(t, 2, model.FormulaTemplate.TestCount)
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

//go:fix inline
func testIntPtr(value int) *int {
	return new(value)
}

//go:fix inline
func testStringPtr(value string) *string {
	return new(value)
}

func TestRatingDetailToModel_MapsEveryChoice(t *testing.T) {
	t.Parallel()

	chosen := "Flat"
	perMile := decimal.RequireFromString("1175")
	flat := decimal.RequireFromString("900.50")
	detail := &shipmentdomain.RatingDetail{
		Choices: []tarifftypes.Choice{{
			Label:    "Linehaul",
			Kind:     tarifftypes.ChoiceBest,
			Currency: "USD",
			Chosen:   &chosen,
			Options: []tarifftypes.ChoiceOption{
				{Label: "Per mile", Status: tarifftypes.OptionLost, Subtotal: &perMile},
				{Label: "Flat", Status: tarifftypes.OptionChosen, Subtotal: &flat},
				{
					Label:   "Volume",
					Status:  tarifftypes.OptionRefused,
					Refusal: "Volume rates start at 6,000 lb",
				},
			},
		}},
	}

	model, err := ratingDetailToModel(detail)
	require.NoError(t, err)
	require.Len(t, model.Choices, 1)
	choice := model.Choices[0]
	assert.Equal(t, "Linehaul", choice.Label)
	assert.Equal(t, "best", choice.Kind)
	assert.Equal(t, "USD", choice.Currency)
	require.NotNil(t, choice.Chosen)
	assert.Equal(t, "Flat", *choice.Chosen)
	require.Len(t, choice.Options, 3)
	assert.Equal(t, "lost", choice.Options[0].Status)
	require.NotNil(t, choice.Options[0].Subtotal)
	assert.Equal(t, "1175", *choice.Options[0].Subtotal)
	assert.Equal(t, "900.5", *choice.Options[1].Subtotal)
	assert.Nil(t, choice.Options[1].Refusal)
	assert.Equal(t, "refused", choice.Options[2].Status)
	assert.Nil(t, choice.Options[2].Subtotal)
	require.NotNil(t, choice.Options[2].Refusal)
	assert.Equal(t, "Volume rates start at 6,000 lb", *choice.Options[2].Refusal)
}

func TestRatingDetailToModel_AnswersNoChoicesAsAnEmptyList(t *testing.T) {
	t.Parallel()

	model, err := ratingDetailToModel(&shipmentdomain.RatingDetail{})
	require.NoError(t, err)
	require.NotNil(t, model.Choices)
	assert.Empty(t, model.Choices)
}

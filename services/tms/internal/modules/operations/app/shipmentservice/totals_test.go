package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/commodityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const perMileSource = `tariff "Per Mile"

input rate: Money/mi

charge "Linehaul" (quantity: shipment.distance, rate: rate)
`

const rollupSource = `tariff "Rollups"

require let weight = shipment.weight else "This rate needs the shipment's weight"

charge "Weight" (quantity: weight, rate: $1/lb)
charge "Linear feet" (quantity: shipment.linearFeet, rate: $1/ft)
charge "Hazmat" = if shipment.hazmat { $100 } else { $0 }
`

const linearFeetSource = `tariff "Linear feet"

charge "Linear feet" (quantity: shipment.linearFeet, rate: $1/ft)
charge "Hazmat" = if shipment.hazmat { $25 } else { $0 }
`

const refusingSource = `tariff "Refuses"

require shipment.hazmat else "Only hazmat loads are rated here"

charge "Linehaul" = $100
`

func totalsTemplate(source string) *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Name:          "Totals",
		Source:        source,
		Inputs: []tarifftypes.InputDecl{{
			Name:       "rate",
			Type:       "Money/mi",
			Descriptor: rules.RawJSON(`{"kind":"quantity","unit":"$/mi"}`),
		}},
	}
}

func tariffCalculator(t *testing.T, template *tariffports.Template) *shipmentcommercial.Calculator {
	t.Helper()

	stack := ratingtest.NewStack(t, ratingtest.StackOptions{Any: template})
	return shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      stack.Engine,
		AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	})
}

func perMileRate(t *testing.T, rate string) tarifftypes.Inputs {
	t.Helper()

	inputs, err := tarifftypes.InputsFromLiterals(map[string]string{"rate": rate})
	require.NoError(t, err)
	return inputs
}

func expectShipmentControl(t *testing.T, entity *shipment.Shipment) *tenantportsmock.MockShipmentControlRepository {
	t.Helper()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	return controlRepo
}

func totalsService(
	t *testing.T,
	entity *shipment.Shipment,
	calculator *shipmentcommercial.Calculator,
) *service {
	t.Helper()

	return &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		controlRepo:  expectShipmentControl(t, entity),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   calculator,
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}
}

func TestServiceCalculateTotals_RatesThroughTheTariffAndAddsCharges(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TariffInputs = perMileRate(t, "$2.50/mi")
	entity.Moves[0].Distance = new(float64(100))
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	svc := totalsService(t, entity, tariffCalculator(t, totalsTemplate(perMileSource)))

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromInt(250).Equal(resp.FreightChargeAmount), "freight %s", resp.FreightChargeAmount)
	assert.True(t, decimal.NewFromInt(10).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(260).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_AShipmentWithNoTariffIsRatedByHand(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.FormulaTemplateID = pulid.Nil
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(500))

	svc := totalsService(t, entity, newTestCommercialCalculator(
		t,
		decimal.NewFromInt(999),
		accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	))

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(500).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(500).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_RejectsNilShipment(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial:   tariffCalculator(t, totalsTemplate(perMileSource)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	resp, err := svc.CalculateTotals(t.Context(), nil, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "shipment")
}

func TestServiceCalculateTotals_CalculatesPerUnitAndPercentageCharges(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.TariffInputs = perMileRate(t, "$3/mi")
	entity.Moves[0].Distance = new(float64(10))
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromInt(5),
			Unit:                2,
		},
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodPercentage,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	svc := totalsService(t, entity, tariffCalculator(t, totalsTemplate(perMileSource)))

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromInt(30).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.NewFromInt(13).Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromInt(43).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_ReportsTheTariffsRefusal(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()

	svc := totalsService(t, entity, tariffCalculator(t, totalsTemplate(refusingSource)))

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.ErrorContains(t, err, "Only hazmat loads are rated here")
}

func TestServiceCalculateTotals_RatesCommodityRollupsThroughTheTariff(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Weight = nil
	entity.Pieces = nil
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: pulid.MustNew("com_"),
			Pieces:      5,
			Weight:      1000,
			Commodity: &commodity.Commodity{
				LinearFeetPerUnit: new(float64(2)),
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{
					ID: pulid.MustNew("hm_"),
				},
			},
		},
		{
			CommodityID: pulid.MustNew("com_"),
			Pieces:      3,
			Weight:      700,
			Commodity: &commodity.Commodity{
				LinearFeetPerUnit: new(1.5),
			},
		},
	}

	calculator := tariffCalculator(t, totalsTemplate(rollupSource))

	svc := totalsService(t, entity, calculator)

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromFloat(1814.5).Equal(resp.FreightChargeAmount))
	assert.True(t, decimal.Zero.Equal(resp.OtherChargeAmount))
	assert.True(t, decimal.NewFromFloat(1814.5).Equal(resp.TotalChargeAmount))
}

func TestServiceCalculateTotals_HydratesCommodityDetailsBeforeRating(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Weight = nil
	entity.Pieces = nil
	firstCommodityID := pulid.MustNew("com_")
	secondCommodityID := pulid.MustNew("com_")
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: firstCommodityID,
			Pieces:      4,
			Weight:      800,
		},
		{
			CommodityID: secondCommodityID,
			Pieces:      2,
			Weight:      200,
		},
	}

	commodityRepo := commodityportsmock.NewMockCommodityRepository(t)
	commodityRepo.EXPECT().
		GetByIDs(mock.Anything, commodityports.GetCommoditiesByIDsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			CommodityIDs: []pulid.ID{
				firstCommodityID,
				secondCommodityID,
			},
		}).
		Return([]*commodity.Commodity{
			{
				ID:                firstCommodityID,
				LinearFeetPerUnit: new(1.25),
				HazardousMaterial: &hazardousmaterial.HazardousMaterial{ID: pulid.MustNew("hm_")},
			},
			{
				ID:                secondCommodityID,
				LinearFeetPerUnit: new(float64(2)),
			},
		}, nil).
		Once()
	calculator := tariffCalculator(t, totalsTemplate(linearFeetSource))

	svc := totalsService(t, entity, calculator)
	svc.commodityRepo = commodityRepo

	resp, err := svc.CalculateTotals(t.Context(), entity, pulid.MustNew("usr_"))

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, decimal.NewFromFloat(34).Equal(resp.FreightChargeAmount))
}

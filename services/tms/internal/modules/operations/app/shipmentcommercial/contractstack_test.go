package shipmentcommercial

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const contractTariff = `tariff "Linehaul with fuel and stops"

/// The rate per mile.
input rate: Money/mi

charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Fuel surcharge" as FSC = $100
charge "Extra stop" as STOP = $50
`

type contractStack struct {
	stack      *ratingtest.Stack
	calculator *Calculator
	template   *tariffports.Template
	rule       *rateagreement.RateAgreementRule
	agreement  *rateagreement.RateAgreement
	stop       *accessorialcharge.AccessorialCharge
	lumper     *accessorialcharge.AccessorialCharge
	layover    *accessorialcharge.AccessorialCharge
	fuel       *accessorialcharge.AccessorialCharge
}

func newContractStack(t *testing.T) *contractStack {
	t.Helper()

	orgID, buID, customerID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("cus_")
	c := &contractStack{
		template: &tariffports.Template{
			TemplateID:    pulid.MustNew("ft_"),
			VersionNumber: 1,
			Name:          "Linehaul with fuel and stops",
			Source:        contractTariff,
			Inputs: []tarifftypes.InputDecl{{
				Name:       "rate",
				Type:       "Money/mi",
				Descriptor: rules.RawJSON(`{"kind":"quantity","unit":"$/mi"}`),
				Required:   true,
			}},
		},
		stop:    &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "STOP"},
		lumper:  &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "LUMP"},
		layover: &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "LAY"},
		fuel:    &accessorialcharge.AccessorialCharge{ID: pulid.MustNew("acc_"), Code: "FSC"},
	}
	catalog := []*accessorialcharge.AccessorialCharge{c.stop, c.lumper, c.layover, c.fuel}

	c.agreement = &rateagreement.RateAgreement{
		ID:             pulid.MustNew("rag_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "ACME-2026",
		Name:           "Acme Foods 2026",
		Currency:       "USD",
		Status:         rateagreement.StatusActive,
		EffectiveFrom:  1,
		Inputs:         tarifftypes.Inputs{},
	}
	c.agreement.Accessorials = []*rateagreement.RateAgreementAccessorial{
		c.accessorial(c.stop, "40", ""),
		c.accessorial(c.lumper, "125", "condition \"Two stops\"\n\nshipment.stops.count() == 2\n"),
		c.accessorial(c.layover, "300", "condition \"Hazmat\"\n\nshipment.hazmat\n"),
	}
	inputs, err := tarifftypes.InputsFromLiterals(map[string]string{"rate": "$2/mi"})
	require.NoError(t, err)
	c.rule = &rateagreement.RateAgreementRule{
		ID:                   pulid.MustNew("ragr_"),
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		RateAgreementID:      c.agreement.ID,
		PartyType:            rateagreement.PartyTypeCustomer,
		PartyID:              customerID,
		Status:               rateagreement.RuleStatusActive,
		Direction:            rateagreement.DirectionDirectional,
		OriginScopeType:      rategeo.ScopeTypeAny,
		DestinationScopeType: rategeo.ScopeTypeAny,
		FormulaTemplateID:    c.template.TemplateID,
		Inputs:               inputs,
		SpecificityScore:     100,
		EffectiveFrom:        1,
		Agreement:            c.agreement,
	}

	agreements := rateagreementportsmock.NewMockRateAgreementRepository(t)
	agreements.EXPECT().
		ResolveRules(mock.Anything, mock.Anything).
		Return(&rateagreementports.ResolveRateRulesResult{
			Rules: []*rateagreement.RateAgreementRule{c.rule},
			Total: 1,
		}, nil).
		Maybe()
	agreements.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(c.agreement, nil).
		Maybe()
	agreements.EXPECT().
		GetRuleByID(mock.Anything, mock.Anything).
		Return(c.rule, nil).
		Maybe()

	c.stack = ratingtest.NewStack(t, ratingtest.StackOptions{
		Templates:  []*tariffports.Template{c.template},
		Agreements: agreements,
		Catalog:    catalog,
	})

	accessorials := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	accessorials.EXPECT().ListCatalog(mock.Anything, mock.Anything).Return(catalog, nil).Maybe()

	fuel := fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t)
	fuel.EXPECT().
		ResolveShipmentCharge(mock.Anything, mock.Anything).
		RunAndReturn(func(
			context.Context,
			*fuelsurchargeports.ResolveShipmentChargeRequest,
		) (*fuelsurchargeports.ResolvedFuelSurcharge, error) {
			return &fuelsurchargeports.ResolvedFuelSurcharge{
				ProgramID:           pulid.MustNew("fsp_"),
				AccessorialChargeID: c.fuel.ID,
				Amount:              decimal.NewFromInt(80),
			}, nil
		}).
		Maybe()

	c.calculator = New(Params{
		Logger:          zap.NewNop(),
		RateEngine:      c.stack.Engine,
		AccessorialRepo: accessorials,
		AgreementRepo:   agreements,
		FuelSurcharge:   fuel,
	})
	c.calculator.now = func() int64 { return 1_700_000_000 }

	return c
}

func (c *contractStack) accessorial(
	charge *accessorialcharge.AccessorialCharge,
	amount string,
	condition string,
) *rateagreement.RateAgreementAccessorial {
	return &rateagreement.RateAgreementAccessorial{
		ID:                  pulid.MustNew("raga_"),
		RateAgreementID:     c.agreement.ID,
		AccessorialChargeID: charge.ID,
		AccessorialCharge:   charge,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.RequireFromString(amount),
		AutoApply:           true,
		Condition:           condition,
	}
}

func (c *contractStack) shipment() *shipment.Shipment {
	return &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    c.rule.OrganizationID,
		BusinessUnitID:    c.rule.BusinessUnitID,
		CustomerID:        c.rule.PartyID,
		Status:            shipment.StatusNew,
		FormulaTemplateID: c.template.TemplateID,
		TariffInputs:      c.rule.Inputs.Clone(),
		Moves: []*shipment.ShipmentMove{{
			Loaded:   true,
			Distance: new(720.0),
			Stops: []*shipment.Stop{
				{
					ID:         pulid.MustNew("stp_"),
					Type:       shipment.StopTypePickup,
					Status:     shipment.StopStatusNew,
					LocationID: pulid.MustNew("loc_"),
				},
				{
					ID:         pulid.MustNew("stp_"),
					Sequence:   1,
					Type:       shipment.StopTypeDelivery,
					Status:     shipment.StopStatusNew,
					LocationID: pulid.MustNew("loc_"),
				},
			},
		}},
	}
}

func chargeFor(
	entity *shipment.Shipment,
	charge *accessorialcharge.AccessorialCharge,
) []*shipment.AdditionalCharge {
	out := make([]*shipment.AdditionalCharge, 0, 1)
	for _, additional := range entity.AdditionalCharges {
		if additional.AccessorialChargeID == charge.ID {
			out = append(out, additional)
		}
	}
	return out
}

func TestRateAndAdoptContract_OneTariffCallPricesTheWholeBill(t *testing.T) {
	t.Parallel()

	c := newContractStack(t)
	entity := c.shipment()

	rating, err := c.calculator.RateAndAdoptContract(
		t.Context(),
		entity,
		&tenant.ShipmentControl{},
		pulid.MustNew("usr_"),
	)
	require.NoError(t, err)
	require.True(t, rating.Adopted)

	assert.EqualValues(t, 1, c.stack.Evaluations(),
		"the freight, the coded charges and every condition come from one call")

	require.Len(t, entity.FreightLines, 1)
	assert.Equal(t, "Linehaul", entity.FreightLines[0].Label)
	assert.True(t, entity.FreightLines[0].Amount.Equal(decimal.NewFromInt(1440)))
	assert.True(t, entity.FreightChargeAmount.Decimal.Equal(decimal.NewFromInt(1440)))

	stops := chargeFor(entity, c.stop)
	require.Len(t, stops, 1, "the tariff owns STOP, so the agreement's row is skipped")
	assert.True(t, stops[0].IsTariff)
	assert.Equal(t, "Extra stop", stops[0].Label)
	assert.True(t, stops[0].Amount.Equal(decimal.NewFromInt(50)))

	fuel := chargeFor(entity, c.fuel)
	require.Len(t, fuel, 1, "the tariff owns FSC, so the fuel program is skipped")
	assert.True(t, fuel[0].IsTariff)
	assert.True(t, fuel[0].Amount.Equal(decimal.NewFromInt(100)))

	lumpers := chargeFor(entity, c.lumper)
	require.Len(t, lumpers, 1, "the two-stop condition held in the same call")
	assert.Equal(t, shipment.SystemOwnerAgreement, lumpers[0].Owner())
	assert.True(t, lumpers[0].Amount.Equal(decimal.NewFromInt(125)))

	assert.Empty(t, chargeFor(entity, c.layover), "the hazmat condition did not hold")

	assert.True(t, entity.TotalChargeAmount.Decimal.Equal(decimal.NewFromInt(1440+50+100+125)),
		"total %s", entity.TotalChargeAmount.Decimal)

	require.NotNil(t, entity.RatingDetail)
	require.NotNil(t, entity.RatingDetail.Bill)
	assert.ElementsMatch(t, []shipment.RatingSkip{
		{Code: "STOP", Source: skipSourceAgreement},
		{Code: "FSC", Source: skipSourceFuel},
	}, entity.RatingDetail.Skipped)
}

func TestRecalculate_ReratesWithOneTariffCallAndKeepsOneOwnerPerCode(t *testing.T) {
	t.Parallel()

	c := newContractStack(t)
	entity := c.shipment()
	userID := pulid.MustNew("usr_")
	control := &tenant.ShipmentControl{}

	_, err := c.calculator.RateAndAdoptContract(t.Context(), entity, control, userID)
	require.NoError(t, err)
	before := c.stack.Evaluations()

	require.NoError(t, c.calculator.Recalculate(t.Context(), entity, control, userID))

	assert.EqualValues(t, 1, c.stack.Evaluations()-before)
	assert.Len(t, chargeFor(entity, c.stop), 1)
	assert.Len(t, chargeFor(entity, c.fuel), 1)
	assert.Len(t, chargeFor(entity, c.lumper), 1)
	assert.True(t, entity.TotalChargeAmount.Decimal.Equal(decimal.NewFromInt(1715)))
}

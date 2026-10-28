package carrierassignmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func chargeIDPtr() *pulid.ID {
	id := pulid.MustNew("acc_")

	return &id
}

func money(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

type buySideFixture struct {
	t         *testing.T
	service   *Service
	catalog   []*accessorialcharge.AccessorialCharge
	fuelIDs   []pulid.ID
	resolver  *fuelsurchargeportsmock.MockFuelSurchargeResolver
	fuelCalls int
}

func newBuySideFixture(t *testing.T) *buySideFixture {
	t.Helper()

	fixture := &buySideFixture{t: t}
	accessorials := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	accessorials.EXPECT().
		ListCatalog(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			_ pagination.TenantInfo,
		) ([]*accessorialcharge.AccessorialCharge, error) {
			return fixture.catalog, nil
		}).
		Maybe()
	fixture.resolver = fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t)
	fixture.resolver.EXPECT().
		FuelChargeIDs(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, pagination.TenantInfo) ([]pulid.ID, error) {
			return fixture.fuelIDs, nil
		}).
		Maybe()
	fixture.service = &Service{accessorialRepo: accessorials, fuelResolver: fixture.resolver}

	return fixture
}

func (f *buySideFixture) charge(code string) *accessorialcharge.AccessorialCharge {
	charge := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        code,
		Description: code + " charge",
	}
	f.catalog = append(f.catalog, charge)

	return charge
}

func (f *buySideFixture) fuel(code string) *accessorialcharge.AccessorialCharge {
	charge := f.charge(code)
	f.fuelIDs = append(f.fuelIDs, charge.ID)

	return charge
}

func (f *buySideFixture) rate(parts *buySideParts) (*buySideResult, error) {
	f.t.Helper()

	if parts.entity == nil {
		parts.entity = &shipment.Shipment{}
	}

	return f.service.buySideRating(f.t.Context(), pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}, parts)
}

func ratedBuySide(amount string) *ratequoteports.RatedShipment {
	quoteID := pulid.MustNew("rqt_")
	agreementID := pulid.MustNew("rag_")

	return &ratequoteports.RatedShipment{
		Amount:      money(amount),
		Currency:    "USD",
		Outcome:     ratequote.OutcomeRated,
		AgreementID: &agreementID,
		Quote: &ratequote.RateQuote{
			ID:      quoteID,
			Outcome: ratequote.OutcomeRated,
		},
	}
}

func billing(
	rated *ratequoteports.RatedShipment,
	lines ...tarifftypes.Line,
) *ratequoteports.RatedShipment {
	rated.Bill = &tarifftypes.Bill{
		Currency: "USD",
		Lines: append(
			[]tarifftypes.Line{{Kind: tarifftypes.LineCharge, Amount: rated.Amount}},
			lines...),
		FreightTotal: rated.Amount,
	}

	return rated
}

func codedLine(code, label, amount string) tarifftypes.Line {
	return tarifftypes.Line{
		Kind:   tarifftypes.LineCharge,
		Code:   code,
		Label:  label,
		Amount: money(amount),
	}
}

func flatAccessorial(
	charge *accessorialcharge.AccessorialCharge,
	amount string,
) *rateagreement.RateAgreementAccessorial {
	return &rateagreement.RateAgreementAccessorial{
		ID:                  pulid.MustNew("raga_"),
		AccessorialChargeID: charge.ID,
		AccessorialCharge:   charge,
		AutoApply:           true,
		Method:              accessorialcharge.MethodFlat,
		Amount:              money(amount),
	}
}

func TestBuySideRating_PricesTheAssignmentAsAFlatAmount(t *testing.T) {
	t.Parallel()

	rated := ratedBuySide("1450.75")
	rating, err := newBuySideFixture(t).rate(&buySideParts{rated: rated})

	require.NoError(t, err)
	assert.Equal(t, shipment.CarrierRateMethodFlat, rating.RateMethod)
	assert.True(t, rating.BaseRate.Equal(money("1450.75")), "base rate was %s", rating.BaseRate)
	require.NotNil(t, rating.RateQuoteID)
	assert.Equal(t, rated.Quote.ID, *rating.RateQuoteID)
	assert.True(t, rating.FuelSurcharge.IsZero())
	assert.Empty(t, rating.Accessorials)
}

func TestBuySideRating_RefusesWhenNoContractCoversTheLane(t *testing.T) {
	t.Parallel()

	rated := ratedBuySide("0")
	rated.Outcome = ratequote.OutcomeNoRateFound

	_, err := newBuySideFixture(t).rate(&buySideParts{rated: rated})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "carrier")
}

func TestBuySideRating_SplitsTheTariffsCodedLinesIntoFuelAndAccessorials(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	fixture.fuel("FSC")
	stop := fixture.charge("STOP")
	rated := billing(
		ratedBuySide("1450"),
		codedLine("fsc", "Fuel", "210"),
		codedLine("STOP", "Extra stop", "75"),
		codedLine("STOP", "Waived stop", "0"),
	)

	rating, err := fixture.rate(&buySideParts{rated: rated})

	require.NoError(t, err)
	assert.True(t, rating.BaseRate.Equal(money("1450")))
	assert.True(t, rating.FuelSurcharge.Equal(money("210")))
	require.Len(t, rating.Accessorials, 1)
	assert.Equal(t, stop.ID, *rating.Accessorials[0].AccessorialChargeID)
	assert.Equal(t, "Extra stop", rating.Accessorials[0].Description)
	assert.True(t, rating.Accessorials[0].Amount.Equal(money("75")))
}

func TestBuySideRating_RefusesACodeTheCatalogDoesNotHave(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	rated := billing(ratedBuySide("1450"), codedLine("LUMP", "Lumper", "90"))

	_, err := fixture.rate(&buySideParts{rated: rated})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "catalog")
}

func TestBuySideRating_TakesAccessorialsFromTheCarriersContract(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	lumper := fixture.charge("LUMP")
	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{flatAccessorial(lumper, "125")},
	}

	rating, err := fixture.rate(&buySideParts{rated: ratedBuySide("1450"), agreement: agreement})

	require.NoError(t, err)
	require.Len(t, rating.Accessorials, 1)
	assert.Equal(t, lumper.ID, *rating.Accessorials[0].AccessorialChargeID)
	assert.Equal(t, "LUMP charge", rating.Accessorials[0].Description)
	assert.True(t, rating.Accessorials[0].Amount.Equal(money("125")))
}

func TestBuySideRating_LeavesManualAccessorialsAlone(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	manual := flatAccessorial(fixture.charge("LUMP"), "125")
	manual.AutoApply = false
	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{manual},
	}

	rating, err := fixture.rate(&buySideParts{rated: ratedBuySide("1450"), agreement: agreement})

	require.NoError(t, err)
	assert.Empty(t, rating.Accessorials)
}

func TestBuySideRating_AppliesAConditionalAccessorialOnlyWhenTheTariffSaidSo(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	met := flatAccessorial(fixture.charge("STOP"), "75")
	met.Condition = "condition \"Multi stop\"\n\nshipment.stops.count() > 2\n"
	unmet := flatAccessorial(fixture.charge("LUMP"), "125")
	unmet.Condition = "condition \"Lumper\"\n\nshipment.stops.count() > 9\n"
	rated := ratedBuySide("1450")
	rated.Conditions = map[pulid.ID]bool{met.ID: true, unmet.ID: false}
	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{met, unmet},
	}

	rating, err := fixture.rate(&buySideParts{rated: rated, agreement: agreement})

	require.NoError(t, err)
	require.Len(t, rating.Accessorials, 1)
	assert.Equal(t, met.AccessorialChargeID, *rating.Accessorials[0].AccessorialChargeID)
}

func TestBuySideRating_SkipsAContractAccessorialTheTariffBilled(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	stop := fixture.charge("STOP")
	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{flatAccessorial(stop, "50")},
	}
	rated := billing(ratedBuySide("1450"), codedLine("STOP", "Extra stop", "75"))

	rating, err := fixture.rate(&buySideParts{rated: rated, agreement: agreement})

	require.NoError(t, err)
	require.Len(t, rating.Accessorials, 1)
	assert.True(t, rating.Accessorials[0].Amount.Equal(money("75")), "the tariff's line wins")
}

func fuelBound(programID pulid.ID) *rateagreement.RateAgreement {
	return &rateagreement.RateAgreement{
		ID: pulid.MustNew("rag_"),
		FuelBinding: &rateagreement.RateAgreementFuelBinding{
			FuelSurchargeProgramID: programID,
		},
	}
}

func TestBuySideRating_PricesFuelFromTheCarrierAgreementsProgram(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	fsc := fixture.fuel("FSC")
	agreement := fuelBound(pulid.MustNew("fsp_"))
	fixture.resolver.EXPECT().
		ResolveShipmentCharge(mock.Anything, mock.MatchedBy(
			func(req *fuelsurchargeports.ResolveShipmentChargeRequest) bool {
				return req.Override != nil && req.Override.AgreementID == agreement.ID &&
					req.Linehaul.Equal(money("1450"))
			},
		)).
		Return(&fuelsurchargeports.ResolvedFuelSurcharge{
			AccessorialChargeID: fsc.ID,
			Amount:              money("188.50"),
		}, nil).
		Once()

	rating, err := fixture.rate(&buySideParts{rated: ratedBuySide("1450"), agreement: agreement})

	require.NoError(t, err)
	assert.True(t, rating.FuelSurcharge.Equal(money("188.50")))
}

func TestBuySideRating_LeavesFuelToTheTariffWhenItBilledFuel(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	fsc := fixture.fuel("FSC")
	fixture.resolver.EXPECT().
		ResolveShipmentCharge(mock.Anything, mock.Anything).
		Return(&fuelsurchargeports.ResolvedFuelSurcharge{
			AccessorialChargeID: fsc.ID,
			Amount:              money("188.50"),
		}, nil).
		Maybe()
	rated := billing(ratedBuySide("1450"), codedLine("FSC", "Fuel", "210"))

	rating, err := fixture.rate(&buySideParts{
		rated:     rated,
		agreement: fuelBound(pulid.MustNew("fsp_")),
	})

	require.NoError(t, err)
	assert.True(t, rating.FuelSurcharge.Equal(money("210")))
}

func TestBuySideRating_AWaivedProgramChargesNoFuel(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	agreement := fuelBound(pulid.MustNew("fsp_"))
	agreement.FuelBinding.Waived = true

	rating, err := fixture.rate(&buySideParts{rated: ratedBuySide("1450"), agreement: agreement})

	require.NoError(t, err)
	assert.True(t, rating.FuelSurcharge.IsZero())
}

func TestBuySideRating_MeasuresMarginAgainstTheWholePay(t *testing.T) {
	t.Parallel()

	fixture := newBuySideFixture(t)
	fixture.fuel("FSC")
	rated := billing(ratedBuySide("1300"), codedLine("FSC", "Fuel", "100"))

	rating, err := fixture.rate(&buySideParts{
		rated:     rated,
		sellTotal: decimal.NewNullDecimal(money("2000")),
	})

	require.NoError(t, err)
	assert.True(t, rating.Margin.Amount.Equal(money("600")),
		"margin was %s", rating.Margin.Amount)
	assert.True(t, rating.Margin.Percent.Equal(money("30")),
		"percent was %s", rating.Margin.Percent)
}

func TestBuySideRating_FlagsAMarginBelowTheContractsFloor(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		MarginFloorPercent: decimal.NewNullDecimal(money("15")),
	}

	rating, err := newBuySideFixture(t).rate(&buySideParts{
		rated:     ratedBuySide("1900"),
		agreement: agreement,
		sellTotal: decimal.NewNullDecimal(money("2000")),
	})

	require.NoError(t, err)
	assert.True(t, rating.Margin.BelowFloor)
}

func TestBuySideRating_UnpricedSellSideIsNotAMarginBreach(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		MarginFloorPercent: decimal.NewNullDecimal(money("15")),
	}

	rating, err := newBuySideFixture(t).rate(&buySideParts{
		rated:     ratedBuySide("1900"),
		agreement: agreement,
	})

	require.NoError(t, err)
	assert.False(t, rating.Margin.Breached())
}

func appliedRequest() *carrierassignmentports.AssignMoveToCarrierRequest {
	return &carrierassignmentports.AssignMoveToCarrierRequest{
		CarrierID: pulid.MustNew("car_"),
		AutoRate:  true,
	}
}

// Auto-rating fills the request the same way a person typing the numbers would,
// so everything downstream — validation, totals, the rate confirmation — runs
// on one path rather than two.
func TestApplyBuySideRating_FillsTheRequestTheWayAPersonWould(t *testing.T) {
	t.Parallel()

	req := appliedRequest()
	quoteID := pulid.MustNew("rqt_")

	applyBuySideRating(req, &buySideResult{
		RateMethod:    shipment.CarrierRateMethodFlat,
		BaseRate:      money("1450"),
		FuelSurcharge: money("210"),
		RateQuoteID:   &quoteID,
		Accessorials: []carrierassignmentports.CarrierAccessorialInput{
			{AccessorialChargeID: chargeIDPtr(), Amount: money("125")},
		},
	})

	assert.Equal(t, shipment.CarrierRateMethodFlat, req.RateMethod)
	assert.True(t, req.BaseRate.Equal(money("1450")))
	assert.True(t, req.FuelSurcharge.Equal(money("210")))
	assert.Len(t, req.Accessorials, 1)
}

// Somebody who typed a rate meant it. Auto-rating over the top of it would be
// the system overruling a negotiated number, which is what makes people switch
// auto-rating off.
func TestApplyBuySideRating_IsNotAskedForWhenARateWasTyped(t *testing.T) {
	t.Parallel()

	req := &carrierassignmentports.AssignMoveToCarrierRequest{
		CarrierID: pulid.MustNew("car_"),
		AutoRate:  true,
		BaseRate:  money("1600"),
	}

	assert.False(t, shouldAutoRate(req), "a typed rate is a decision, not a gap")
}

func TestShouldAutoRate_IsSkippedWhenNotAskedFor(t *testing.T) {
	t.Parallel()

	assert.False(t, shouldAutoRate(&carrierassignmentports.AssignMoveToCarrierRequest{}))
}

func TestShouldAutoRate_AppliesWhenAskedForAndNoRateWasTyped(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldAutoRate(appliedRequest()))
}

// A guardrail the organization turned on has to actually stop the assignment,
// or it is decoration.
func TestEnforceMarginFloor_BlocksABreachWhenEnforcementIsOn(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{
			BelowFloor:  true,
			Explanation: "margin of 5% is below the 15% this contract requires",
		},
		true,
		false,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "margin terms")
	assert.NotContains(t, err.Error(), "%",
		"the refusal reaches whoever assigns the carrier, so it states no margin")
}

// With enforcement off the breach is still worth knowing about, but it is not
// the system's decision to make.
func TestEnforceMarginFloor_AllowsABreachWhenEnforcementIsOff(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{BelowFloor: true, Explanation: "below floor"},
		false,
		false,
	)

	assert.NoError(t, err)
}

// The override is how somebody says "I know, do it anyway", mirroring the
// insurance warning this service already has.
func TestEnforceMarginFloor_AnExplicitOverrideGoesThrough(t *testing.T) {
	t.Parallel()

	err := enforceMarginFloor(
		ratetypes.MarginVerdict{BelowFloor: true, Explanation: "below floor"},
		true,
		true,
	)

	assert.NoError(t, err)
}

func TestEnforceMarginFloor_NothingBreachedNeedsNoOverride(t *testing.T) {
	t.Parallel()

	assert.NoError(t, enforceMarginFloor(ratetypes.MarginVerdict{}, true, false))
}

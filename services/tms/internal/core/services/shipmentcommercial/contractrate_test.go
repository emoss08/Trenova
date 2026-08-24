package shipmentcommercial

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A recalculation runs from seven call sites, and every one of them used to
// re-resolve the contract. That is what silently replaced a spot rate somebody
// had negotiated, so the routine path now prices through the shipment's own
// formula template and asks for no contract at all.
func TestRecalculate_PricesFormulaOnlyAndPersistsNoQuote(t *testing.T) {
	t.Parallel()

	entity := validShipment()

	var captured *services.RateShipmentRequest

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *services.RateShipmentRequest,
		) (*services.RatedShipment, error) {
			captured = req

			return &services.RatedShipment{
				Amount:   decimal.NewFromInt(1000),
				Currency: "USD",
				Outcome:  ratequote.OutcomeFormulaFallback,
			}, nil
		}).
		Once()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	require.NotNil(t, captured)
	assert.True(t, captured.FormulaOnly, "a recalculation must not resolve a contract")
	assert.False(t, captured.Persist, "a keystroke is not a rating decision worth recording")
}

// The contract stamped on the shipment has to survive a formula-only rating,
// or the billing panel starts naming a formula where the rater expects to see
// the agreement and concludes the contract stopped applying.
func TestRecalculate_KeepsTheContractProvenanceOnAnAutoRatedShipment(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.AutoRated = true
	entity.RatingDetail = &shipment.RatingDetail{
		AgreementID:   "ragr_01",
		AgreementName: "Acme TL 2026",
		RuleLabel:     "Dallas to Chicago",
		RateQuoteID:   "rq_01",
	}

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	require.NotNil(t, entity.RatingDetail)
	assert.Equal(t, "Acme TL 2026", entity.RatingDetail.AgreementName)
	assert.Equal(t, "Dallas to Chicago", entity.RatingDetail.RuleLabel)
	assert.Equal(t, "rq_01", entity.RatingDetail.RateQuoteID)
}

// A shipment somebody has priced by hand carries no contract, so carrying the
// contract's name onto it would claim an agreement priced something it did not.
func TestRecalculate_DropsTheContractProvenanceOnceEdited(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.AutoRated = false
	entity.RatingDetail = &shipment.RatingDetail{AgreementName: "Acme TL 2026"}

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	require.NotNil(t, entity.RatingDetail)
	assert.Empty(t, entity.RatingDetail.AgreementName)
}

// An invoiced shipment has numbers the customer has already seen. Nothing about
// it is recomputed, and the rate engine is not consulted at all.
func TestRecalculate_LockedShipmentKeepsItsCharges(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.RateLocked = true
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(1234))

	engine := mocks.NewMockRateEngine(t)

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	assert.True(t, decimal.NewFromInt(1234).Equal(entity.FreightChargeAmount.Decimal))
}

// Adopting a contract is what auto-rating is: the agreement hands over a rating
// method and a base rate, and they become the shipment's own fields.
func TestAdoptContractRate_SeatsTheRatingMethodAndBaseRate(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.BaseRate = decimal.NewNullDecimal(decimal.NewFromInt(1))

	templateID := pulid.MustNew("fmt_")
	agreementID := pulid.MustNew("ragr_")

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	calculator.AdoptContractRate(t.Context(), entity, &services.RatedShipment{
		Amount:            decimal.NewFromInt(1000),
		Outcome:           ratequote.OutcomeRated,
		AgreementID:       &agreementID,
		FormulaTemplateID: &templateID,
		BaseRate:          decimal.NewNullDecimal(decimal.NewFromFloat(2.15)),
	})

	assert.Equal(t, templateID, entity.FormulaTemplateID)
	assert.True(t, decimal.NewFromFloat(2.15).Equal(entity.BaseRate.Decimal))
	assert.True(t, entity.AutoRated)
	require.NotNil(t, entity.AutoRatedAt)
	assert.Equal(t, ratedAt, *entity.AutoRatedAt)
}

// A rule that binds no rate of its own prices through whatever the shipment
// already carries, so there is nothing to seat and overwriting would erase a
// figure the contract deliberately deferred to.
func TestAdoptContractRate_KeepsTheBaseRateWhenTheRuleBindsNone(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(3.5))

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	calculator.AdoptContractRate(t.Context(), entity, &services.RatedShipment{
		Amount:  decimal.NewFromInt(1000),
		Outcome: ratequote.OutcomeRated,
	})

	assert.True(t, decimal.NewFromFloat(3.5).Equal(entity.BaseRate.Decimal))
}

// A fresh application of the contract makes any previous departure untrue.
func TestAdoptContractRate_ClearsAPreviousDeparture(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	entity := validShipment()
	entity.RateOverrideAmount = decimal.NewNullDecimal(decimal.NewFromInt(900))
	entity.RateOverrideReason = "Spot rate agreed with the customer"
	entity.RateOverrideByID = &userID

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	calculator.AdoptContractRate(t.Context(), entity, &services.RatedShipment{
		Amount:  decimal.NewFromInt(1000),
		Outcome: ratequote.OutcomeRated,
	})

	assert.False(t, entity.HasRateOverride())
	assert.Empty(t, entity.RateOverrideReason)
	assert.Nil(t, entity.RateOverrideByID)
}

// Nothing is adopted from a lane no contract covers, or the shipment would be
// stamped as contract-rated at whatever the fallback produced.
func TestAdoptContractRate_IgnoresAnUnpricedOutcome(t *testing.T) {
	t.Parallel()

	entity := validShipment()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 0)})
	calculator.now = func() int64 { return ratedAt }

	calculator.AdoptContractRate(t.Context(), entity, &services.RatedShipment{
		Amount:  decimal.Zero,
		Outcome: ratequote.OutcomeNoRateFound,
	})

	assert.False(t, entity.AutoRated)
}

// The rating method and the base rate are compared, not the resulting money:
// the money follows from them and from the mileage, and a rounding difference
// in the total is not somebody having renegotiated the rate.
func TestContractRateMatches(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("fmt_")
	otherTemplateID := pulid.MustNew("fmt_")

	rated := &services.RatedShipment{
		Outcome:           ratequote.OutcomeRated,
		FormulaTemplateID: &templateID,
		BaseRate:          decimal.NewNullDecimal(decimal.NewFromFloat(2.15)),
	}

	matching := validShipment()
	matching.FormulaTemplateID = templateID
	matching.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(2.15))
	assert.True(t, ContractRateMatches(matching, rated))

	repriced := validShipment()
	repriced.FormulaTemplateID = templateID
	repriced.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(1.95))
	assert.False(t, ContractRateMatches(repriced, rated))

	remethoded := validShipment()
	remethoded.FormulaTemplateID = otherTemplateID
	remethoded.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(2.15))
	assert.False(t, ContractRateMatches(remethoded, rated))

	assert.False(t, ContractRateMatches(matching, &services.RatedShipment{
		Outcome: ratequote.OutcomeNoRateFound,
	}))
}

// The difference between what was charged and what the contract said is the
// rate leakage report, and it only stays true if it is stored when it happens.
func TestRecordRateDeparture_RecordsWhatWasGivenAway(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	quoteID := pulid.MustNew("rq_")

	entity := validShipment()
	entity.AutoRated = false
	entity.RateQuoteID = &quoteID
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(900))

	contractQuote := &ratequote.RateQuote{
		ID:             quoteID,
		Outcome:        ratequote.OutcomeRated,
		Currency:       "USD",
		LinehaulAmount: decimal.NewFromInt(1000),
		TotalAmount:    decimal.NewFromInt(1000),
	}

	var recorded *ratequote.RateQuote

	quoteRepo := mocks.NewMockRateQuoteRepository(t)
	quoteRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateQuoteByIDRequest")).
		Return(contractQuote, nil).
		Once()
	quoteRepo.EXPECT().
		Record(mock.Anything, mock.AnythingOfType("*ratequote.RateQuote")).
		RunAndReturn(func(
			_ context.Context,
			quote *ratequote.RateQuote,
		) (*ratequote.RateQuote, error) {
			recorded = quote
			quote.ID = pulid.MustNew("rq_")

			return quote, nil
		}).
		Once()

	calculator := New(Params{
		Logger:     zap.NewNop(),
		RateEngine: StubRateEngine(t, 1000),
		QuoteRepo:  quoteRepo,
	})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(t, calculator.RecordRateDeparture(t.Context(), entity, userID, true))

	require.True(t, entity.RateOverrideAmount.Valid)
	assert.True(t, decimal.NewFromInt(900).Equal(entity.RateOverrideAmount.Decimal))
	require.NotNil(t, entity.RateOverrideByID)
	assert.Equal(t, userID, *entity.RateOverrideByID)

	require.NotNil(t, recorded)
	assert.Equal(t, ratequote.OutcomeManualOverride, recorded.Outcome)
	assert.Equal(t, "USD", recorded.Currency)
	require.True(t, recorded.ForegoneAmount.Valid)
	assert.True(t, decimal.NewFromInt(100).Equal(recorded.ForegoneAmount.Decimal),
		"the contract wanted 1000 and we billed 900, so 100 was given away")
}

// Later edits to an already departed shipment are not a second departure. A
// quote for each of them would turn the rating history into a keystroke log.
func TestRecordRateDeparture_WritesNoQuoteWhenNothingJustDeparted(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.AutoRated = false
	entity.RateOverrideAmount = decimal.NewNullDecimal(decimal.NewFromInt(900))
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(850))

	quoteRepo := mocks.NewMockRateQuoteRepository(t)

	calculator := New(Params{
		Logger:     zap.NewNop(),
		RateEngine: StubRateEngine(t, 1000),
		QuoteRepo:  quoteRepo,
	})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(t, calculator.RecordRateDeparture(t.Context(), entity, pulid.MustNew("usr_"), false))

	assert.True(t, decimal.NewFromInt(850).Equal(entity.RateOverrideAmount.Decimal),
		"the recorded amount tracks what is actually charged")
}

// A shipment still carrying its contract rate has departed from nothing.
func TestRecordRateDeparture_IgnoresAnAutoRatedShipment(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.AutoRated = true

	quoteRepo := mocks.NewMockRateQuoteRepository(t)

	calculator := New(Params{
		Logger:     zap.NewNop(),
		RateEngine: StubRateEngine(t, 1000),
		QuoteRepo:  quoteRepo,
	})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(t, calculator.RecordRateDeparture(t.Context(), entity, pulid.MustNew("usr_"), true))

	assert.False(t, entity.HasRateOverride())
}

// A rater who negotiates the linehaul has not renegotiated the schedule. The
// agreement still covers the lane, so the charges it applies automatically are
// still owed — dropping them under-bills a load whose rate was merely haggled
// over, and the panel showed them before the save.
func TestRateAndAdoptContract_KeepsTheContractAccessorialsOnADeparture(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))
	entity := validShipment()
	entity.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(1.95))

	calculator := departingCalculator(t, agreement, decimal.NewFromFloat(2.15))

	rating, err := calculator.RateAndAdoptContract(
		t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_"),
	)
	require.NoError(t, err)
	require.False(t, rating.Adopted, "a hand-priced base rate is a departure")

	require.Len(t, entity.AdditionalCharges, 1)
	charge := entity.AdditionalCharges[0]
	assert.Equal(t, shipment.SystemOwnerAgreement, charge.Owner())
	assert.Equal(t, agreement.Accessorials[0].ID, *charge.RateAgreementAccessorialID)
	assert.True(t, decimal.NewFromInt(75).Equal(charge.Amount))
	assert.True(t, decimal.NewFromInt(75).Equal(entity.OtherChargeAmount.Decimal))

	assert.False(t, entity.AutoRated, "the rate departed, so the shipment does not carry the contract's")
}

// No contract covering the lane means no schedule to apply. A shipment priced
// by the organization's fallback formula must not be handed charges from an
// agreement that never spoke for it.
func TestRateAndAdoptContract_AddsNoAccessorialsWhenNothingPricedTheLane(t *testing.T) {
	t.Parallel()

	agreement := agreementWithAccessorial(t, decimal.NewFromInt(75))

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:   decimal.Zero,
			Currency: "USD",
			Outcome:  ratequote.OutcomeNoRateFound,
		}, nil).
		Maybe()

	calculator := New(Params{
		Logger:        zap.NewNop(),
		RateEngine:    engine,
		AgreementRepo: agreementRepoFor(t, agreement),
	})
	calculator.now = func() int64 { return ratedAt }

	entity := validShipment()

	rating, err := calculator.RateAndAdoptContract(
		t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_"),
	)
	require.NoError(t, err)
	require.False(t, rating.Adopted)

	assert.Empty(t, entity.AdditionalCharges)
}

// departingCalculator prices through a contract that binds a base rate, so a
// shipment carrying a different one is a departure rather than an adoption.
func departingCalculator(
	t *testing.T,
	agreement *rateagreement.RateAgreement,
	contractBaseRate decimal.Decimal,
) *Calculator {
	t.Helper()

	linehaul := decimal.NewFromInt(100)
	agreementID := agreement.ID

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:      linehaul,
			Currency:    "USD",
			Outcome:     ratequote.OutcomeRated,
			AgreementID: &agreementID,
			BaseRate:    decimal.NewNullDecimal(contractBaseRate),
			Quote: &ratequote.RateQuote{
				Outcome:         ratequote.OutcomeRated,
				RateAgreementID: &agreementID,
				LinehaulAmount:  linehaul,
				TotalAmount:     linehaul,
				RatedAt:         ratedAt,
			},
		}, nil).
		Maybe()

	calculator := New(Params{
		Logger:        zap.NewNop(),
		RateEngine:    engine,
		AgreementRepo: agreementRepoFor(t, agreement),
	})
	calculator.now = func() int64 { return ratedAt }

	return calculator
}

func agreementRepoFor(
	t *testing.T,
	agreement *rateagreement.RateAgreement,
) *mocks.MockRateAgreementRepository {
	t.Helper()

	repo := mocks.NewMockRateAgreementRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetRateAgreementByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			if req.RateAgreementID != agreement.ID {
				return nil, errors.New("unexpected agreement id")
			}

			return agreement, nil
		}).
		Maybe()

	return repo
}

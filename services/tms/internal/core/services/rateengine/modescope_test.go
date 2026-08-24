package rateengine

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubModeProfiles answers with one policy, or with the ordinary "nothing
// configured" outcome.
type stubModeProfiles struct {
	services.ModeProfileService

	policy *modeprofile.ResolvedPolicy
	err    error
	calls  int
}

func (s *stubModeProfiles) Resolve(
	_ context.Context,
	_ *services.ResolveModeProfileRequest,
) (*modeprofile.ResolvedPolicy, error) {
	s.calls++

	if s.err != nil {
		return nil, s.err
	}

	return s.policy, nil
}

func drayageRule(rate string) *rateagreement.RateAgreementRule {
	rule := perMileRule(rate, 100)
	rule.Label = "drayage-only"
	rule.ServiceModels = []modeprofile.ServiceModel{modeprofile.ServiceModelDrayage}

	return rule
}

// A drayage tariff and a truckload tariff on the same lane is the ordinary case
// for anybody running both, and the mode is the only thing telling them apart.
// Without the shipment's own mode reaching the resolver, the drayage rule can
// never match and its lane silently prices at the truckload rate — or at
// nothing at all.
func TestModeScopedRuleMatchesTheShipmentsResolvedMode(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = &stubModeProfiles{
		policy: &modeprofile.ResolvedPolicy{
			ServiceModel:   modeprofile.ServiceModelDrayage,
			EquipmentClass: modeprofile.EquipmentClassContainer,
		},
	}

	expectRules(d, drayageRule("3.05"))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
	// 720 mi at $3.05 is $2,196.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("2196")),
		"amount was %s", result.Amount)
}

// The inverse has to hold too, or the scope means nothing: a truckload shipment
// must not be priced by the drayage tariff.
func TestModeScopedRuleIsRejectedWhenTheModeDiffers(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = &stubModeProfiles{
		policy: &modeprofile.ResolvedPolicy{
			ServiceModel:   modeprofile.ServiceModelTruckload,
			EquipmentClass: modeprofile.EquipmentClassDryVan,
		},
	}

	expectRules(d, drayageRule("3.05"))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeNoRateFound, result.Outcome)

	require.NotNil(t, result.Quote)
	require.NotNil(t, result.Quote.Trace)
	require.Len(t, result.Quote.Trace.Candidates, 1)
	assert.Equal(t,
		ratetypes.RejectReasonServiceModelMismatch,
		result.Quote.Trace.Candidates[0].RejectReason)
}

// Capability enforcement is opt-in, so an organization with no profiles is not
// in error. Rating has to keep working for them, which means an unscoped rule
// still prices the shipment.
func TestNoModeProfileConfiguredStillPricesAnUnscopedRule(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = &stubModeProfiles{err: services.ErrNoModeProfileConfigured}

	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1548")),
		"amount was %s", result.Amount)
}

// A mode lookup that fails for any other reason must not take the rating down
// with it. Pricing at the unscoped rate is a smaller problem than a shipment
// that cannot be saved because a capability service was briefly unavailable.
func TestModeLookupFailureDoesNotFailTheRating(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = &stubModeProfiles{
		err: errortypes.NewBusinessError("mode profile service is unavailable"),
	}

	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
}

// Rating happens on every shipment write, so the mode is resolved once per
// rating rather than once per rule considered.
func TestModeIsResolvedOncePerRating(t *testing.T) {
	t.Parallel()

	d := setup(t)
	stub := &stubModeProfiles{
		policy: &modeprofile.ResolvedPolicy{ServiceModel: modeprofile.ServiceModelDrayage},
	}
	d.svc.modeProfile = stub

	expectRules(d, drayageRule("3.05"), drayageRule("2.95"), drayageRule("2.85"))

	_, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, 1, stub.calls)
}

// The mode a rating used belongs in the quote. A dispute months later asks why
// the drayage tariff applied, and "the shipment resolved to drayage" is the
// answer — which is only available if it was written down.
func TestQuoteRecordsTheModeItRatedUnder(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = &stubModeProfiles{
		policy: &modeprofile.ResolvedPolicy{ServiceModel: modeprofile.ServiceModelDrayage},
	}

	expectRules(d, drayageRule("3.05"))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	require.NotNil(t, result.Quote)
	require.NotNil(t, result.Quote.Trace)
	assert.Equal(t,
		string(modeprofile.ServiceModelDrayage),
		result.Quote.Trace.Inputs.ServiceModel)
}

// An engine wired without the mode service at all — a test harness, a path that
// predates this — must not panic. It rates as though nothing is configured.
func TestMissingModeServiceBehavesAsNothingConfigured(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.svc.modeProfile = nil

	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
}

// The formula template a shipment carries is the customer's rating method. On
// the buy side, falling back to it would pay the carrier whatever the customer
// is charged — a load sold at $2,000 would settle at $2,000 and the margin
// would be exactly nothing.
//
// A carrier with no contract has no rate. That is the honest answer, and it is
// the one that makes somebody go and write the contract.
func TestBuySideDoesNotFallBackToTheShipmentsFormulaTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	entity := testShipment()
	entity.FormulaTemplateID = pulid.MustNew("ft_")

	result, err := d.svc.RateShipment(t.Context(), &services.RateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo(),
		PartyType:  rateagreement.PartyTypeCarrier,
		PartyID:    pulid.MustNew("car_"),
		Purpose:    ratequote.PurposeRating,
	})

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeNoRateFound, result.Outcome)
	assert.True(t, result.Amount.IsZero(), "amount was %s", result.Amount)
	d.formula.AssertNotCalled(t, "Calculate", mock.Anything, mock.Anything)
}

// The sell side keeps the fallback, because that is what makes adopting rate
// agreements safe: until a customer contract exists, every shipment rates
// exactly as it did before.
func TestSellSideStillFallsBackToTheShipmentsFormulaTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	templateID := pulid.MustNew("ft_")
	entity := testShipment()
	entity.FormulaTemplateID = templateID

	d.formula.EXPECT().Calculate(mock.Anything, mock.Anything).
		Return(&formulatemplatetypes.CalculateResponse{
			Amount:            decimal.RequireFromString("1500"),
			FormulaTemplateID: templateID.String(),
		}, nil)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(entity))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeFormulaFallback, result.Outcome)
}

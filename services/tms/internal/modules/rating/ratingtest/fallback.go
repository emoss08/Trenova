package ratingtest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratematrixportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratezoneportsmock"

	"github.com/emoss08/trenova/internal/modules/rating/app/rateengine"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// NewFallbackEngine returns a rate engine backed by the given formula
// calculator and by contract storage that covers nothing.
//
// It exists for the tests of everything downstream of rating — the shipment
// service, the totals endpoint, the state coordinator — which care that a
// shipment ends up with the right numbers, not which contract produced them.
// Resolving nothing puts the engine on its fallback path, so those tests go on
// asserting exactly what they asserted before rate agreements existed.
func NewFallbackEngine(t *testing.T, formula formulatemplateports.FormulaCalculator) *rateengine.Service {
	t.Helper()

	agreementRepo := rateagreementportsmock.NewMockRateAgreementRepository(t)
	agreementRepo.EXPECT().
		ResolveRules(mock.Anything, mock.AnythingOfType("*rateagreementports.ResolveRateRulesRequest")).
		Return(&rateagreementports.ResolveRateRulesResult{}, nil).
		Maybe()

	zoneRepo := ratezoneportsmock.NewMockRateZoneRepository(t)
	zoneRepo.EXPECT().
		ResolveMembership(
			mock.Anything,
			mock.AnythingOfType("*ratezoneports.ResolveZoneMembershipRequest"),
		).
		Return(nil, nil).
		Maybe()

	quoteRepo := ratequoteportsmock.NewMockRateQuoteRepository(t)
	quoteRepo.EXPECT().
		Record(mock.Anything, mock.AnythingOfType("*ratequote.RateQuote")).
		RunAndReturn(func(
			_ context.Context,
			quote *ratequote.RateQuote,
		) (*ratequote.RateQuote, error) {
			if quote.ID.IsNil() {
				quote.ID = pulid.MustNew("rqt_")
			}
			return quote, nil
		}).
		Maybe()

	return rateengine.New(rateengine.Params{
		Logger:        zap.NewNop(),
		AgreementRepo: agreementRepo,
		ZoneRepo:      zoneRepo,
		MatrixRepo:    ratematrixportsmock.NewMockRateMatrixRepository(t),
		DensityRepo:   ratematrixportsmock.NewMockDensityScaleRepository(t),
		QuoteRepo:     quoteRepo,
		Formula:       formula,
		Exchange:      exchangerateportsmock.NewMockExchangeRateService(t),
	})
}

package rateengine

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
func NewFallbackEngine(t *testing.T, formula services.FormulaCalculator) *Service {
	t.Helper()

	agreementRepo := mocks.NewMockRateAgreementRepository(t)
	agreementRepo.EXPECT().
		ResolveRules(mock.Anything, mock.AnythingOfType("*repositories.ResolveRateRulesRequest")).
		Return(&repositories.ResolveRateRulesResult{}, nil).
		Maybe()

	zoneRepo := mocks.NewMockRateZoneRepository(t)
	zoneRepo.EXPECT().
		ResolveMembership(
			mock.Anything,
			mock.AnythingOfType("*repositories.ResolveZoneMembershipRequest"),
		).
		Return(nil, nil).
		Maybe()

	quoteRepo := mocks.NewMockRateQuoteRepository(t)
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

	return New(Params{
		Logger:        zap.NewNop(),
		AgreementRepo: agreementRepo,
		ZoneRepo:      zoneRepo,
		MatrixRepo:    mocks.NewMockRateMatrixRepository(t),
		DensityRepo:   mocks.NewMockDensityScaleRepository(t),
		QuoteRepo:     quoteRepo,
		Formula:       formula,
		Exchange:      mocks.NewMockExchangeRateService(t),
	})
}

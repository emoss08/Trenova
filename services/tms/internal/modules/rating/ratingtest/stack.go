package ratingtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/rateengine"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type Stack struct {
	Engine     *rateengine.Service
	Rater      *tariff.Rater
	Templates  *tarifftest.Templates
	Context    *tarifftest.StaticContext
	Evaluator  *tarifftest.Evaluator
	Agreements *rateagreementportsmock.MockRateAgreementRepository
}

func (s *Stack) Evaluations() int64 {
	return s.Evaluator.Calls()
}

type StackOptions struct {
	Templates  []*tariffports.Template
	Any        *tariffports.Template
	Context    *rules.TariffContext
	Hydrator   ratequoteports.ShipmentRatingHydrator
	Agreements *rateagreementportsmock.MockRateAgreementRepository
	Catalog    []*accessorialcharge.AccessorialCharge
}

type NoHydration struct{}

func (NoHydration) HydrateShipment(
	context.Context,
	*ratequoteports.HydrateShipmentRequest,
) error {
	return nil
}

func (NoHydration) HydrateShipments(
	context.Context,
	*ratequoteports.HydrateShipmentsRequest,
) error {
	return nil
}

type noModeProfile struct {
	modeprofileports.ModeProfileService
}

func (noModeProfile) Resolve(
	context.Context,
	*modeprofileports.ResolveModeProfileRequest,
) (*modeprofile.ResolvedPolicy, error) {
	return nil, modeprofileports.ErrNoModeProfileConfigured
}

func NewStack(t *testing.T, opts StackOptions) *Stack {
	t.Helper()

	hydrator := opts.Hydrator
	if hydrator == nil {
		hydrator = NoHydration{}
	}

	stack := &Stack{
		Templates: tarifftest.NewTemplates(opts.Templates...),
		Context:   &tarifftest.StaticContext{Base: opts.Context},
		Evaluator: &tarifftest.Evaluator{},
	}
	if opts.Any != nil {
		stack.Templates.Fallback(opts.Any)
	}
	stack.Rater = tarifftest.NewRater(stack.Context, stack.Evaluator)

	agreements := opts.Agreements
	if agreements == nil {
		agreements = rateagreementportsmock.NewMockRateAgreementRepository(t)
		agreements.EXPECT().
			ResolveRules(mock.Anything, mock.AnythingOfType("*rateagreementports.ResolveRateRulesRequest")).
			Return(&rateagreementports.ResolveRateRulesResult{}, nil).
			Maybe()
	}
	stack.Agreements = agreements

	quotes := ratequoteportsmock.NewMockRateQuoteRepository(t)
	quotes.EXPECT().
		Record(mock.Anything, mock.AnythingOfType("*ratequote.RateQuote")).
		RunAndReturn(func(_ context.Context, quote *ratequote.RateQuote) (*ratequote.RateQuote, error) {
			if quote.ID.IsNil() {
				quote.ID = pulid.MustNew("rqt_")
			}
			return quote, nil
		}).
		Maybe()

	fuel := fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t)
	fuel.EXPECT().FuelChargeIDs(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	accessorials := accessorialchargeportsmock.NewMockAccessorialChargeRepository(t)
	accessorials.EXPECT().
		ListCatalog(mock.Anything, mock.Anything).
		Return(opts.Catalog, nil).
		Maybe()

	stack.Engine = rateengine.New(rateengine.Params{
		Logger:        zap.NewNop(),
		AgreementRepo: agreements,
		QuoteRepo:     quotes,
		Resolver:      stack.Templates,
		Rater:         stack.Rater,
		Records:       tarifftest.Records{},
		Fuel:          fuel,
		Accessorials:  accessorials,
		Exchange:      exchangerateportsmock.NewMockExchangeRateService(t),
		ModeProfile:   noModeProfile{},
		Hydrator:      hydrator,
	})

	return stack
}

func FlatTemplate(amount decimal.Decimal) *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Name:          "Flat",
		Source: fmt.Sprintf(
			"tariff \"Flat\"\n\ncharge \"Freight\" = $%s\n",
			amount.StringFixed(2),
		),
	}
}

func NewEngine(t *testing.T) *rateengine.Service {
	t.Helper()
	return NewStack(t, StackOptions{}).Engine
}

func NewFlatEngine(t *testing.T, amount decimal.Decimal) *rateengine.Service {
	t.Helper()
	return NewStack(t, StackOptions{Any: FlatTemplate(amount)}).Engine
}

func NewHydratingFlatEngine(
	t *testing.T,
	amount decimal.Decimal,
	hydrator ratequoteports.ShipmentRatingHydrator,
) *rateengine.Service {
	t.Helper()
	return NewStack(t, StackOptions{Any: FlatTemplate(amount), Hydrator: hydrator}).Engine
}

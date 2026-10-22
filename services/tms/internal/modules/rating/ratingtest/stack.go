package ratingtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateengine"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type Templates struct {
	mu        sync.Mutex
	templates map[pulid.ID]*tariffports.Template
	any       *tariffports.Template
}

func NewTemplates(templates ...*tariffports.Template) *Templates {
	store := &Templates{templates: make(map[pulid.ID]*tariffports.Template, len(templates))}
	for _, template := range templates {
		store.Add(template)
	}
	return store
}

func (t *Templates) Add(template *tariffports.Template) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.templates[template.TemplateID] = template
}

func (t *Templates) ResolveBilling(
	_ context.Context,
	req *tariffports.ResolveRequest,
) (*tariffports.Template, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	template, ok := t.templates[req.TemplateID]
	if !ok && t.any != nil {
		template, ok = t.any, true
	}
	if !ok {
		return nil, errortypes.NewNotFoundError("Formula template not found")
	}
	copied := *template
	return &copied, nil
}

type StaticContext struct {
	Base *rules.TariffContext
}

func (s *StaticContext) Context(
	_ context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	base := s.Base
	if base == nil {
		base = &rules.TariffContext{}
	}
	copied := *base
	copied.Tables = append([]rules.TariffTable(nil), base.Tables...)
	copied.Libraries = req.Libraries
	return &copied, nil
}

func (s *StaticContext) LoadCells(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]rules.TariffNeed,
) error {
	return nil
}

func (s *StaticContext) LoadTables(
	context.Context,
	pagination.TenantInfo,
	*rules.TariffContext,
	[]string,
) error {
	return nil
}

type Records struct{}

func (Records) Build(_ context.Context, req *tariff.RecordRequest) (rules.RawJSON, error) {
	origin := record.PlaceOf(req.Shipment.ShipperStop())
	destination := record.PlaceOf(req.Shipment.DeliveryStop())
	origin.Zones, destination.Zones = req.OriginZones, req.DestinationZones
	return record.Build(&record.Input{
		Shipment:     req.Shipment,
		Origin:       origin,
		Destination:  destination,
		DistanceUnit: record.DistanceMiles,
		Rating: record.Rating{
			At:        time.Unix(req.AsOf, 0).UTC(),
			Mode:      req.Mode,
			SellTotal: req.SellTotal,
			Currency:  req.Currency,
		},
	})
}

func (Records) LaneZones(
	context.Context,
	pagination.TenantInfo,
	*shipment.Shipment,
) (*ratezoneports.LaneZones, error) {
	return &ratezoneports.LaneZones{}, nil
}

type Stack struct {
	Engine      *rateengine.Service
	Rater       *tariff.Rater
	Templates   *Templates
	Context     *StaticContext
	Agreements  *rateagreementportsmock.MockRateAgreementRepository
	evaluations atomic.Int64
}

func (s *Stack) Evaluations() int64 {
	return s.evaluations.Load()
}

type StackOptions struct {
	Templates  []*tariffports.Template
	Any        *tariffports.Template
	Context    *rules.TariffContext
	Hydrator   ratequoteports.ShipmentRatingHydrator
	Agreements *rateagreementportsmock.MockRateAgreementRepository
}

func NewStack(t *testing.T, opts StackOptions) *Stack {
	t.Helper()

	stack := &Stack{
		Templates: NewTemplates(opts.Templates...),
		Context:   &StaticContext{Base: opts.Context},
	}
	stack.Templates.any = opts.Any
	stack.Rater = tariff.NewRaterWith(
		zap.NewNop(),
		stack.Context,
		func(ctx context.Context, req *rules.TariffEvaluateRequest) (*rules.TariffEvaluation, error) {
			stack.evaluations.Add(1)
			return rules.EvaluateTariffs(ctx, req)
		},
	)

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

	stack.Engine = rateengine.New(rateengine.Params{
		Logger:        zap.NewNop(),
		AgreementRepo: agreements,
		QuoteRepo:     quotes,
		Resolver:      stack.Templates,
		Rater:         stack.Rater,
		Records:       Records{},
		Fuel:          fuel,
		Accessorials:  accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		Exchange:      exchangerateportsmock.NewMockExchangeRateService(t),
		Hydrator:      opts.Hydrator,
	})

	return stack
}

func FlatTemplate(amount decimal.Decimal) *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    pulid.MustNew("ft_"),
		VersionNumber: 1,
		Name:          "Flat",
		Source:        fmt.Sprintf("tariff \"Flat\"\n\ncharge \"Freight\" = $%s\n", amount.StringFixed(2)),
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

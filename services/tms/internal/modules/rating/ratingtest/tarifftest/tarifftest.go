package tarifftest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const PerMileSource = `tariff "Per Mile"

/// The rate per mile.
input rate: Money/mi
/// The least the shipment pays.
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)

minimum minimumCharge
`

func PerMile() *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    pulid.ID("ft_per_mile_stub"),
		VersionNumber: 3,
		Name:          "Per Mile",
		Source:        PerMileSource,
		Inputs: []tarifftypes.InputDecl{
			{
				Name: "rate",
				Type: "Money/mi",
				Descriptor: rules.RawJSON(
					`{"dimension":"Money/Distance","kind":"quantity","unit":"$/mi"}`,
				),
				Required: true,
			},
			{
				Name:       "minimumCharge",
				Type:       "Money",
				Descriptor: rules.RawJSON(`{"kind":"money"}`),
				Default:    rules.MoneyValue(decimal.Zero),
			},
		},
	}
}

type Templates struct {
	mu        sync.Mutex
	templates map[pulid.ID]*tariffports.Template
	fallback  *tariffports.Template
	resolves  atomic.Int64
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

func (t *Templates) Fallback(template *tariffports.Template) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fallback = template
}

func (t *Templates) Resolves() int64 {
	return t.resolves.Load()
}

func (t *Templates) ResolveBilling(
	_ context.Context,
	req *tariffports.ResolveRequest,
) (*tariffports.Template, error) {
	t.resolves.Add(1)
	t.mu.Lock()
	defer t.mu.Unlock()
	template, ok := t.templates[req.TemplateID]
	if !ok && t.fallback != nil {
		template, ok = t.fallback, true
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

type Records struct {
	Zones *ratezoneports.LaneZones
	Fuel  record.Fuel
}

func (r Records) Build(_ context.Context, req *tariff.RecordRequest) (rules.RawJSON, error) {
	origin := record.PlaceOf(req.Shipment.ShipperStop())
	destination := record.PlaceOf(req.Shipment.DeliveryStop())
	origin.Zones, destination.Zones = req.OriginZones, req.DestinationZones
	return record.Build(&record.Input{
		Shipment:     req.Shipment,
		Origin:       origin,
		Destination:  destination,
		Fuel:         r.Fuel,
		DistanceUnit: record.DistanceMiles,
		Rating: record.Rating{
			At:        time.Unix(req.AsOf, 0).UTC(),
			Mode:      req.Mode,
			SellTotal: req.SellTotal,
			Currency:  req.Currency,
		},
	})
}

func (r Records) LaneZones(
	context.Context,
	pagination.TenantInfo,
	*shipment.Shipment,
) (*ratezoneports.LaneZones, error) {
	if r.Zones != nil {
		return r.Zones, nil
	}
	return &ratezoneports.LaneZones{}, nil
}

type Evaluator struct {
	calls    atomic.Int64
	requests []*rules.TariffEvaluateRequest
	mu       sync.Mutex
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	req *rules.TariffEvaluateRequest,
) (*rules.TariffEvaluation, error) {
	e.calls.Add(1)
	e.mu.Lock()
	e.requests = append(e.requests, req)
	e.mu.Unlock()
	return rules.EvaluateTariffs(ctx, req)
}

func (e *Evaluator) Calls() int64 {
	return e.calls.Load()
}

func (e *Evaluator) Requests() []*rules.TariffEvaluateRequest {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]*rules.TariffEvaluateRequest(nil), e.requests...)
}

func NewRater(contexts tariffports.ContextSource, evaluator *Evaluator) *tariff.Rater {
	return tariff.NewRaterWith(zap.NewNop(), contexts, evaluator.Evaluate)
}

type Checker struct {
	calls atomic.Int64
}

func (c *Checker) Calls() int64 {
	return c.calls.Load()
}

func (c *Checker) Check(
	ctx context.Context,
	req *tariffports.CheckRequest,
) (*rules.TariffCheck, error) {
	c.calls.Add(1)
	return rules.CheckTariff(ctx, &rules.TariffCheckRequest{
		Source:  req.Source,
		Kind:    req.Kind,
		Context: &rules.TariffContext{Libraries: req.Libraries},
		Values:  req.Values,
	})
}

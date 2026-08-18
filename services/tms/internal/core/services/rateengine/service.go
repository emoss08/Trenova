package rateengine

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	AgreementRepo repositories.RateAgreementRepository
	ZoneRepo      repositories.RateZoneRepository
	MatrixRepo    repositories.RateMatrixRepository
	DensityRepo   repositories.DensityScaleRepository
	QuoteRepo     repositories.RateQuoteRepository
	Formula       services.FormulaCalculator
	Exchange      services.ExchangeRateService
	ModeProfile   services.ModeProfileService
	RoutingGuides repositories.RoutingGuideRepository
}

type Service struct {
	l             *zap.Logger
	agreementRepo repositories.RateAgreementRepository
	zoneRepo      repositories.RateZoneRepository
	matrixRepo    repositories.RateMatrixRepository
	densityRepo   repositories.DensityScaleRepository
	quoteRepo     repositories.RateQuoteRepository
	formula       services.FormulaCalculator
	exchange      services.ExchangeRateService
	modeProfile   services.ModeProfileService
	routingGuides repositories.RoutingGuideRepository
	now           func() int64
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.rateengine"),
		agreementRepo: p.AgreementRepo,
		zoneRepo:      p.ZoneRepo,
		matrixRepo:    p.MatrixRepo,
		densityRepo:   p.DensityRepo,
		quoteRepo:     p.QuoteRepo,
		formula:       p.Formula,
		exchange:      p.Exchange,
		modeProfile:   p.ModeProfile,
		routingGuides: p.RoutingGuides,
		now:           timeutils.NowUnix,
	}
}

// RateShipment prices one side of a shipment and records why.
//
// A quote is written whatever happens — including when the lane fell through to
// a formula template and when nothing covered it at all. The cases worth
// explaining are exactly the ones nobody predicted, and a record that only
// exists on the happy path explains nothing.
func (s *Service) RateShipment(
	ctx context.Context,
	req *services.RateShipmentRequest,
) (*services.RatedShipment, error) {
	rateCtx, err := s.buildContext(ctx, req)
	if err != nil {
		return nil, err
	}

	log := s.l.With(
		zap.String("operation", "RateShipment"),
		zap.String("partyType", string(rateCtx.PartyType)),
		zap.String("partyId", rateCtx.PartyID.String()),
	)

	trace := &ratetypes.Trace{
		EngineVersion: EngineVersion,
		Inputs:        rateCtx.Inputs(),
	}

	if err = s.resolveZones(ctx, rateCtx); err != nil {
		log.Error("failed to resolve zone membership", zap.Error(err))
		return nil, err
	}

	// The inputs are re-read after zones are known, so the trace records the
	// zones the lane was actually looked up under rather than an empty list.
	trace.Inputs = rateCtx.Inputs()

	resolution, err := s.resolve(ctx, rateCtx)
	if err != nil {
		return nil, err
	}

	trace.LaneKeysTried = resolution.LaneKeysTried
	trace.CandidateCount = resolution.CandidateCount
	trace.Candidates = resolution.Candidates
	trace.TieBreak = resolution.TieBreak

	if resolution.Capped {
		trace.Warn(
			"More rates matched this lane than could be compared; " +
				"overlapping contracts should be narrowed",
		)
	}

	if resolution.Winner == nil {
		return s.rateWithoutAgreement(ctx, req, rateCtx, trace)
	}

	return s.rateFromAgreement(ctx, req, rateCtx, resolution, trace)
}

func (s *Service) rateFromAgreement(
	ctx context.Context,
	req *services.RateShipmentRequest,
	rateCtx *RateContext,
	resolution *Resolution,
	trace *ratetypes.Trace,
) (*services.RatedShipment, error) {
	winner := resolution.Winner

	priced, err := s.price(ctx, rateCtx, winner, trace)
	if err != nil {
		// A rule that cannot be priced is worth recording rather than throwing
		// away: it is nearly always a contract that needs fixing, and the quote
		// is where somebody will look for it.
		trace.Error = err.Error()

		return s.finish(ctx, req, rateCtx, trace, &quoteFields{
			Outcome:     ratequote.OutcomeError,
			Amount:      decimal.Zero,
			Currency:    rateCtx.BillingCurrency,
			AgreementID: &winner.RateAgreementID,
			RuleID:      &winner.ID,
			Specificity: winner.SpecificityScore,
			Agreement:   winner.Agreement,
		})
	}

	converted := s.convert(ctx, rateCtx, priced, trace)

	trace.Totals = totalsFor(converted)

	fields := &quoteFields{
		Outcome:     ratequote.OutcomeRated,
		Amount:      converted,
		Currency:    priced.Currency,
		AgreementID: &winner.RateAgreementID,
		RuleID:      &winner.ID,
		Specificity: winner.SpecificityScore,
		Agreement:   winner.Agreement,
	}

	if winner.RatingBasis == rateagreement.RatingBasisFormula {
		fields.FormulaTemplateID = winner.FormulaTemplateID
	}

	return s.finish(ctx, req, rateCtx, trace, fields)
}

// rateWithoutAgreement handles a lane no contract covers.
//
// What happens is the organization's decision, and the default is the one that
// changes nothing: fall back to the formula template the shipment already
// carried. That is what makes adopting rate agreements safe — until a contract
// exists, every shipment rates exactly as it did before.
func (s *Service) rateWithoutAgreement(
	ctx context.Context,
	req *services.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
) (*services.RatedShipment, error) {
	// The buy side never falls back to a formula template. The one a shipment
	// carries is the customer's rating method, so falling back to it would pay
	// the carrier whatever the customer is charged: a load sold at $2,000 would
	// settle at $2,000 and the margin would be exactly nothing. A carrier with
	// no contract has no rate, and saying so is what makes somebody go and
	// write the contract.
	if rateCtx.PartyType == rateagreement.PartyTypeCarrier {
		trace.Warn("No carrier agreement covers this lane")
		trace.Totals = totalsFor(decimal.Zero)

		return s.finish(ctx, req, rateCtx, trace, &quoteFields{
			Outcome:  ratequote.OutcomeNoRateFound,
			Amount:   decimal.Zero,
			Currency: rateCtx.BillingCurrency,
		})
	}

	disposition := tenant.UnratedShipmentDispositionFallbackFormulaTemplate
	if req.BillingControl != nil && req.BillingControl.UnratedShipmentDisposition != "" {
		disposition = req.BillingControl.UnratedShipmentDisposition
	}

	switch disposition { //nolint:exhaustive // the fallback disposition is the default arm
	case tenant.UnratedShipmentDispositionBlock:
		return nil, errortypes.NewBusinessError(
			"No rate agreement covers this lane",
		)

	case tenant.UnratedShipmentDispositionZeroAndFlag:
		trace.Warn("No rate agreement covers this lane")
		trace.Totals = totalsFor(decimal.Zero)

		return s.finish(ctx, req, rateCtx, trace, &quoteFields{
			Outcome:  ratequote.OutcomeNoRateFound,
			Amount:   decimal.Zero,
			Currency: rateCtx.BillingCurrency,
		})

	default:
		return s.rateFromFallbackTemplate(ctx, req, rateCtx, trace)
	}
}

func (s *Service) rateFromFallbackTemplate(
	ctx context.Context,
	req *services.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
) (*services.RatedShipment, error) {
	templateID := fallbackTemplateID(req)

	if templateID.IsNil() {
		trace.Warn("No rate agreement covers this lane and no formula template was set")
		trace.Totals = totalsFor(decimal.Zero)

		return s.finish(ctx, req, rateCtx, trace, &quoteFields{
			Outcome:  ratequote.OutcomeNoRateFound,
			Amount:   decimal.Zero,
			Currency: rateCtx.BillingCurrency,
		})
	}

	resp, err := s.formula.Calculate(ctx, &formulatemplatetypes.CalculateRequest{
		TemplateID: templateID,
		Entity:     rateCtx.Entity,
		TenantInfo: rateCtx.TenantInfo,
		RatingDate: rateCtx.AsOf,
	})
	if err != nil {
		// A template that will not evaluate has always failed the save, and it
		// keeps failing it. This is the path an organization with no contracts
		// takes for every shipment, and quietly turning a broken formula into a
		// zero-dollar load would be a change in behaviour nobody asked for.
		// A rule inside a contract is treated differently — see rateFromAgreement
		// — because there a failure is recorded and flagged rather than allowed
		// to block every shipment on the lane.
		return nil, err
	}

	trace.AddComponent(&ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      linehaulLabel,
		Basis:      resp.FormulaTemplateName,
		Amount:     resp.Amount,
		Source:     ratetypes.ComponentSourceFormulaTemplate,
		SourceID:   resp.FormulaTemplateID,
		SourceName: resp.FormulaTemplateName,
		Detail:     map[string]any{"expression": resp.Expression},
	})
	trace.Totals = totalsFor(resp.Amount)

	return s.finish(ctx, req, rateCtx, trace, &quoteFields{
		Outcome:           ratequote.OutcomeFormulaFallback,
		Amount:            resp.Amount,
		Currency:          rateCtx.BillingCurrency,
		FormulaTemplateID: &templateID,
	})
}

// fallbackTemplateID prefers the template on the shipment over the
// organization's, because a template chosen for this load is a more specific
// statement of intent than a default chosen for all of them.
func fallbackTemplateID(req *services.RateShipmentRequest) pulid.ID {
	if req.Shipment != nil && !req.Shipment.FormulaTemplateID.IsNil() {
		return req.Shipment.FormulaTemplateID
	}

	if req.BillingControl != nil && req.BillingControl.FallbackFormulaTemplateID != nil {
		return *req.BillingControl.FallbackFormulaTemplateID
	}

	return pulid.Nil
}

// quoteFields is what varies between the outcomes; everything else on a quote
// is assembled the same way whatever happened.
type quoteFields struct {
	Outcome           ratequote.Outcome
	Amount            decimal.Decimal
	Currency          string
	AgreementID       *pulid.ID
	RuleID            *pulid.ID
	FormulaTemplateID *pulid.ID
	Specificity       int32

	// Agreement is the contract that priced this, carried so a fresh quote
	// exposes the same relation one read back from the database does. A caller
	// reading a negotiated term off the quote should not get nothing merely
	// because the quote has not been round tripped yet.
	Agreement *rateagreement.RateAgreement
}

func (s *Service) finish(
	ctx context.Context,
	req *services.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
	fields *quoteFields,
) (*services.RatedShipment, error) {
	quote := s.buildQuote(req, rateCtx, trace, fields)

	if req.Persist {
		recorded, err := s.quoteRepo.Record(ctx, quote)
		if err != nil {
			return nil, err
		}

		// The stored row comes back without its relations loaded, and the
		// contract is what a caller reads negotiated terms from.
		if recorded != nil && recorded.Agreement == nil {
			recorded.Agreement = quote.Agreement
		}

		quote = recorded
	}

	return &services.RatedShipment{
		Amount:            fields.Amount,
		Currency:          rateCtx.BillingCurrency,
		Outcome:           fields.Outcome,
		Quote:             quote,
		AgreementID:       fields.AgreementID,
		RuleID:            fields.RuleID,
		FormulaTemplateID: fields.FormulaTemplateID,
	}, nil
}

func (s *Service) buildQuote(
	req *services.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
	fields *quoteFields,
) *ratequote.RateQuote {
	purpose := req.Purpose
	if purpose == "" {
		purpose = ratequote.PurposeRating
	}

	status := ratequote.StatusQuoted
	if purpose == ratequote.PurposeRating && req.Shipment != nil {
		status = ratequote.StatusApplied
	}

	quote := &ratequote.RateQuote{
		OrganizationID:      rateCtx.TenantInfo.OrgID,
		BusinessUnitID:      rateCtx.TenantInfo.BuID,
		PartyType:           rateCtx.PartyType,
		PartyID:             rateCtx.PartyID,
		Purpose:             purpose,
		Outcome:             fields.Outcome,
		Status:              status,
		RateAgreementID:     fields.AgreementID,
		RateAgreementRuleID: fields.RuleID,
		FormulaTemplateID:   fields.FormulaTemplateID,
		SpecificityScore:    fields.Specificity,
		Agreement:           fields.Agreement,
		Currency:            defaultString(fields.Currency, rateCtx.BillingCurrency),
		LinehaulAmount:      fields.Amount,
		TotalAmount:         fields.Amount,
		BillingCurrency:     rateCtx.BillingCurrency,
		BillingAmount:       fields.Amount,
		AsOf:                rateCtx.AsOf,
		RatedAt:             s.now(),
		EngineVersion:       EngineVersion,
		ContextHash:         rateCtx.Hash(),
		Trace:               trace,
	}

	if req.Shipment != nil {
		shipmentID := req.Shipment.ID
		quote.ShipmentID = &shipmentID
	}

	if !req.UserID.IsNil() {
		userID := req.UserID
		quote.RatedByID = &userID
	}

	if trace.FX != nil {
		quote.FxRate = decimal.NewNullDecimal(trace.FX.Rate)
	}

	return quote
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

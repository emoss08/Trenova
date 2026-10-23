package rateengine

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Records interface {
	Build(ctx context.Context, req *tariff.RecordRequest) (rules.RawJSON, error)
	LaneZones(
		ctx context.Context,
		tenant pagination.TenantInfo,
		entity *shipment.Shipment,
	) (*ratezoneports.LaneZones, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	AgreementRepo rateagreementports.RateAgreementRepository
	QuoteRepo     ratequoteports.RateQuoteRepository
	Resolver      tariffports.Resolver
	Rater         tariffports.Rater
	Records       Records
	Fuel          fuelsurchargeports.FuelSurchargeResolver
	Accessorials  accessorialchargeports.AccessorialChargeRepository
	Exchange      exchangerateports.ExchangeRateService
	ModeProfile   modeprofileports.ModeProfileService
	RoutingGuides tenderports.RoutingGuideRepository
	Hydrator      ratequoteports.ShipmentRatingHydrator
}

type Service struct {
	l             *zap.Logger
	agreementRepo rateagreementports.RateAgreementRepository
	quoteRepo     ratequoteports.RateQuoteRepository
	resolver      tariffports.Resolver
	rater         tariffports.Rater
	records       Records
	fuel          fuelsurchargeports.FuelSurchargeResolver
	accessorials  accessorialchargeports.AccessorialChargeRepository
	exchange      exchangerateports.ExchangeRateService
	modeProfile   modeprofileports.ModeProfileService
	routingGuides tenderports.RoutingGuideRepository
	hydrator      ratequoteports.ShipmentRatingHydrator
	now           func() int64
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.rateengine"),
		agreementRepo: p.AgreementRepo,
		quoteRepo:     p.QuoteRepo,
		resolver:      p.Resolver,
		rater:         p.Rater,
		records:       p.Records,
		fuel:          p.Fuel,
		accessorials:  p.Accessorials,
		exchange:      p.Exchange,
		modeProfile:   p.ModeProfile,
		routingGuides: p.RoutingGuides,
		hydrator:      p.Hydrator,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) RateShipment(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
) (*ratequoteports.RatedShipment, error) {
	if err := s.hydrate(ctx, req.Shipment, req.TenantInfo); err != nil {
		return nil, err
	}

	rateCtx, err := s.buildContext(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = s.resolveZones(ctx, rateCtx); err != nil {
		return nil, err
	}

	trace := &ratetypes.Trace{
		EngineVersion: EngineVersion,
		Inputs:        rateCtx.Inputs(),
	}

	if req.FormulaOnly {
		return s.rateOwnTariff(ctx, req, rateCtx, trace)
	}

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

	return s.rateFromAgreement(ctx, req, rateCtx, resolution.Winner, trace)
}

func (s *Service) rateFromAgreement(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
	rateCtx *RateContext,
	winner *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (*ratequoteports.RatedShipment, error) {
	agreement, err := s.loadAgreement(ctx, rateCtx.TenantInfo, winner.RateAgreementID)
	if err != nil {
		return nil, err
	}
	if agreement == nil {
		agreement = winner.Agreement
	}

	fields := &quoteFields{
		Currency:          winner.EffectiveCurrency(agreement),
		AgreementID:       &winner.RateAgreementID,
		RuleID:            &winner.ID,
		Specificity:       winner.SpecificityScore,
		Agreement:         agreement,
		FormulaTemplateID: &winner.FormulaTemplateID,
	}

	result, err := s.price(ctx, rateCtx, &pricing{
		templateID: winner.FormulaTemplateID,
		inputs:     resolvedInputs(winner, agreement),
		currency:   fields.Currency,
		agreement:  agreement,
		rule:       winner,
	}, trace)
	if err != nil {
		trace.Error = errortypes.LocalizedText(ctx, err)
		fields.Outcome = ratequote.OutcomeError
		return s.finish(ctx, req, rateCtx, trace, fields)
	}

	fields.apply(result)
	if result.refusal != "" {
		trace.Error = result.refusal
		fields.Outcome = ratequote.OutcomeError
		return s.finish(ctx, req, rateCtx, trace, fields)
	}

	fields.Outcome = ratequote.OutcomeRated
	return s.finish(ctx, req, rateCtx, trace, fields)
}

func (s *Service) rateOwnTariff(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
) (*ratequoteports.RatedShipment, error) {
	entity := req.Shipment
	if !entity.AutoRated || entity.RateAgreementID == nil || entity.RateAgreementID.IsNil() {
		return s.rateFromFallbackTemplate(ctx, req, rateCtx, trace)
	}

	agreement, err := s.loadAgreement(ctx, rateCtx.TenantInfo, *entity.RateAgreementID)
	if err != nil {
		return nil, err
	}
	rule := s.loadRule(ctx, rateCtx.TenantInfo, entity.RateAgreementRuleID)

	templateID := fallbackTemplateID(req)
	fields := &quoteFields{
		Currency:          currencyOf(rule, agreement),
		AgreementID:       entity.RateAgreementID,
		RuleID:            entity.RateAgreementRuleID,
		Agreement:         agreement,
		FormulaTemplateID: &templateID,
	}
	result, err := s.price(ctx, rateCtx, &pricing{
		templateID: templateID,
		inputs:     entity.TariffInputs,
		currency:   fields.Currency,
		agreement:  agreement,
		rule:       rule,
	}, trace)
	if err != nil {
		return nil, err
	}
	if result.refusal != "" {
		return nil, refusalError(result.refusal, rule)
	}

	fields.apply(result)
	fields.Outcome = ratequote.OutcomeRated
	return s.finish(ctx, req, rateCtx, trace, fields)
}

func (s *Service) rateWithoutAgreement(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
) (*ratequoteports.RatedShipment, error) {
	if rateCtx.PartyType == rateagreement.PartyTypeCarrier {
		trace.Warn("No carrier agreement covers this lane")
		return s.finish(
			ctx,
			req,
			rateCtx,
			trace,
			&quoteFields{Outcome: ratequote.OutcomeNoRateFound},
		)
	}

	disposition := tenant.UnratedShipmentDispositionFallbackFormulaTemplate
	if req.BillingControl != nil && req.BillingControl.UnratedShipmentDisposition != "" {
		disposition = req.BillingControl.UnratedShipmentDisposition
	}

	switch disposition { //nolint:exhaustive // the fallback disposition is the default arm
	case tenant.UnratedShipmentDispositionBlock:
		return nil, errortypes.NewBusinessError("No rate agreement covers this lane")

	case tenant.UnratedShipmentDispositionZeroAndFlag:
		trace.Warn("No rate agreement covers this lane")
		return s.finish(
			ctx,
			req,
			rateCtx,
			trace,
			&quoteFields{Outcome: ratequote.OutcomeNoRateFound},
		)

	default:
		return s.rateFromFallbackTemplate(ctx, req, rateCtx, trace)
	}
}

func (s *Service) rateFromFallbackTemplate(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
) (*ratequoteports.RatedShipment, error) {
	templateID := fallbackTemplateID(req)
	if templateID.IsNil() {
		trace.Warn("No rate agreement covers this lane and no formula template was set")
		return s.finish(
			ctx,
			req,
			rateCtx,
			trace,
			&quoteFields{Outcome: ratequote.OutcomeNoRateFound},
		)
	}

	var inputs tarifftypes.Inputs
	if req.Shipment != nil && req.Shipment.FormulaTemplateID == templateID {
		inputs = req.Shipment.TariffInputs
	}

	result, err := s.price(ctx, rateCtx, &pricing{
		templateID: templateID,
		inputs:     inputs,
		currency:   rateCtx.BillingCurrency,
	}, trace)
	if err != nil {
		return nil, err
	}
	if result.refusal != "" {
		return nil, refusalError(result.refusal, nil)
	}

	fields := &quoteFields{
		Outcome:           ratequote.OutcomeFormulaFallback,
		Currency:          rateCtx.BillingCurrency,
		FormulaTemplateID: &templateID,
	}
	fields.apply(result)
	return s.finish(ctx, req, rateCtx, trace, fields)
}

func fallbackTemplateID(req *ratequoteports.RateShipmentRequest) pulid.ID {
	if req.Shipment != nil && !req.Shipment.FormulaTemplateID.IsNil() {
		return req.Shipment.FormulaTemplateID
	}

	if req.BillingControl != nil && req.BillingControl.FallbackFormulaTemplateID != nil {
		return *req.BillingControl.FallbackFormulaTemplateID
	}

	return pulid.Nil
}

func (s *Service) loadAgreement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	agreementID pulid.ID,
) (*rateagreement.RateAgreement, error) {
	agreement, err := s.agreementRepo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: agreementID,
		TenantInfo:      tenantInfo,
		IncludeChildren: true,
		OmitRules:       true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // an agreement removed since its rule was resolved prices nothing
		}
		return nil, err
	}
	return agreement, nil
}

func (s *Service) loadRule(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ruleID *pulid.ID,
) *rateagreement.RateAgreementRule {
	if ruleID == nil || ruleID.IsNil() {
		return nil
	}
	rule, err := s.agreementRepo.GetRuleByID(
		ctx,
		&rateagreementports.GetRateAgreementRuleByIDRequest{
			RuleID:     *ruleID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		s.l.Warn(
			"failed to load the rule a shipment adopted",
			zap.String("ruleId", ruleID.String()),
			zap.Error(err),
		)
		return nil
	}
	return rule
}

func currencyOf(
	rule *rateagreement.RateAgreementRule,
	agreement *rateagreement.RateAgreement,
) string {
	if rule != nil {
		return rule.EffectiveCurrency(agreement)
	}
	if agreement != nil {
		return agreement.Currency
	}
	return ""
}

type quoteFields struct {
	Outcome           ratequote.Outcome
	Bill              *tarifftypes.Bill
	Refusal           string
	Choices           []tarifftypes.Choice
	Currency          string
	AgreementID       *pulid.ID
	RuleID            *pulid.ID
	FormulaTemplateID *pulid.ID
	TemplateName      string
	TemplateVersion   int64
	Specificity       int32
	Inputs            tarifftypes.Inputs
	Conditions        map[pulid.ID]bool
	Agreement         *rateagreement.RateAgreement
}

func (f *quoteFields) apply(result *priced) {
	f.Bill = result.bill
	f.Refusal = result.refusal
	f.Choices = result.choices
	f.Inputs = result.inputs
	f.Conditions = result.conditions
	if result.template != nil {
		id := result.template.TemplateID
		f.FormulaTemplateID = &id
		f.TemplateName = result.template.Name
		f.TemplateVersion = result.template.VersionNumber
	}
}

func (s *Service) finish(
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
	rateCtx *RateContext,
	trace *ratetypes.Trace,
	fields *quoteFields,
) (*ratequoteports.RatedShipment, error) {
	totals, err := s.totals(ctx, rateCtx.TenantInfo, fields.Bill)
	if err != nil {
		return nil, err
	}
	trace.Totals = totals

	quote := s.buildQuote(req, rateCtx, trace, fields)

	if req.Persist {
		recorded, recordErr := s.quoteRepo.Record(ctx, quote)
		if recordErr != nil {
			return nil, recordErr
		}
		if recorded != nil && recorded.Agreement == nil {
			recorded.Agreement = quote.Agreement
		}
		quote = recorded
	}

	return &ratequoteports.RatedShipment{
		Amount:              totals.Linehaul,
		Total:               totals.Total,
		Currency:            rateCtx.BillingCurrency,
		Outcome:             fields.Outcome,
		Quote:               quote,
		AgreementID:         fields.AgreementID,
		RuleID:              fields.RuleID,
		BillToCustomerID:    agreementBillTo(fields.Agreement),
		FormulaTemplateID:   fields.FormulaTemplateID,
		FormulaTemplateName: fields.TemplateName,
		TemplateVersion:     fields.TemplateVersion,
		TariffInputs:        fields.Inputs,
		Bill:                fields.Bill,
		Refusal:             fields.Refusal,
		Choices:             fields.Choices,
		Agreement:           fields.Agreement,
		Conditions:          fields.Conditions,
	}, nil
}

func (s *Service) totals(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	bill *tarifftypes.Bill,
) (ratetypes.Totals, error) {
	if bill == nil {
		return ratetypes.Totals{}, nil
	}
	totals := ratetypes.Totals{Linehaul: bill.FreightTotal, Total: bill.Total}
	coded := bill.CodedLines()
	if len(coded) == 0 {
		return totals, nil
	}
	fuelCodes, err := s.fuelCodes(ctx, tenantInfo)
	if err != nil {
		return ratetypes.Totals{}, err
	}
	for i := range coded {
		if _, ok := fuelCodes[accessorialcharge.NormalizeCode(coded[i].Code)]; ok {
			totals.Fuel = totals.Fuel.Add(coded[i].Amount)
			continue
		}
		totals.Accessorial = totals.Accessorial.Add(coded[i].Amount)
	}
	return totals, nil
}

func (s *Service) fuelCodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]struct{}, error) {
	ids, err := s.fuel.FuelChargeIDs(ctx, tenantInfo)
	if err != nil || len(ids) == 0 {
		return map[string]struct{}{}, err
	}
	catalog, err := s.accessorials.ListCatalog(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	codes := make(map[string]struct{}, len(ids))
	for _, charge := range catalog {
		if _, ok := wanted[charge.ID]; ok {
			codes[accessorialcharge.NormalizeCode(charge.Code)] = struct{}{}
		}
	}
	return codes, nil
}

func (s *Service) buildQuote(
	req *ratequoteports.RateShipmentRequest,
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
		LinehaulAmount:      trace.Totals.Linehaul,
		FuelAmount:          trace.Totals.Fuel,
		AccessorialAmount:   trace.Totals.Accessorial,
		TotalAmount:         trace.Totals.Total,
		BillingCurrency:     rateCtx.BillingCurrency,
		BillingAmount:       trace.Totals.Total,
		AsOf:                rateCtx.AsOf,
		RatedAt:             s.now(),
		EngineVersion:       EngineVersion,
		ContextHash:         rateCtx.Hash(),
		Trace:               trace,
	}
	if fields.TemplateVersion > 0 {
		version := fields.TemplateVersion
		quote.FormulaTemplateVersionNumber = &version
	}

	if req.Shipment != nil && req.Shipment.ID.IsNotNil() {
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

func agreementBillTo(agreement *rateagreement.RateAgreement) *pulid.ID {
	if agreement == nil || agreement.BillToCustomerID == nil || agreement.BillToCustomerID.IsNil() {
		return nil
	}
	id := *agreement.BillToCustomerID

	return &id
}

func (s *Service) hydrate(
	ctx context.Context,
	entity *shipment.Shipment,
	tenantInfo pagination.TenantInfo,
) error {
	if s.hydrator == nil || entity == nil {
		return nil
	}

	return s.hydrator.HydrateShipment(ctx, &ratequoteports.HydrateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo,
	})
}

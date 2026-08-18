//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentcommercial

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	RateEngine         services.RateEngine
	Predicate          services.FormulaPredicateEvaluator
	AccessorialRepo    repositories.AccessorialChargeRepository
	AgreementRepo      repositories.RateAgreementRepository
	BillingControlRepo repositories.BillingControlRepository
	FuelSurcharge      services.FuelSurchargeResolver
	DetentionEngine    *detentionservice.Service
}

type Calculator struct {
	logger             *zap.Logger
	rateEngine         services.RateEngine
	predicate          services.FormulaPredicateEvaluator
	accessorialRepo    repositories.AccessorialChargeRepository
	agreementRepo      repositories.RateAgreementRepository
	billingControlRepo repositories.BillingControlRepository
	fuelSurcharge      services.FuelSurchargeResolver
	detentionEngine    *detentionservice.Service
	now                func() int64
}

func New(p Params) *Calculator {
	return &Calculator{
		logger:             p.Logger.Named("service.shipmentcommercial"),
		rateEngine:         p.RateEngine,
		predicate:          p.Predicate,
		accessorialRepo:    p.AccessorialRepo,
		agreementRepo:      p.AgreementRepo,
		billingControlRepo: p.BillingControlRepo,
		fuelSurcharge:      p.FuelSurcharge,
		detentionEngine:    p.DetentionEngine,
		now:                timeutils.NowUnix,
	}
}

type chargeSyncOptions struct {
	detention bool
	fuel      bool
}

func (c *Calculator) Recalculate(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) error {
	baseCharge, otherChargeAmount, ratingDetail, err := c.calculateCommercialTotals(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{detention: true, fuel: true},
	)
	if err != nil {
		return err
	}

	entity.FreightChargeAmount = decimal.NewNullDecimal(baseCharge)
	entity.OtherChargeAmount = decimal.NewNullDecimal(otherChargeAmount)
	entity.TotalChargeAmount = decimal.NewNullDecimal(baseCharge.Add(otherChargeAmount))
	entity.RatingDetail = ratingDetail

	return nil
}

func (c *Calculator) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	baseCharge, otherChargeAmount, _, err := c.calculateCommercialTotals(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{fuel: true},
	)
	if err != nil {
		return nil, err
	}

	return &repositories.ShipmentTotalsResponse{
		FreightChargeAmount: baseCharge,
		OtherChargeAmount:   otherChargeAmount,
		TotalChargeAmount:   baseCharge.Add(otherChargeAmount),
		FuelSurcharge:       findGeneratedFuelSurchargeCharge(entity),
	}, nil
}

func findGeneratedFuelSurchargeCharge(entity *shipment.Shipment) *shipment.AdditionalCharge {
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil {
			return charge
		}
	}
	return nil
}

func CalculateAdditionalCharges(
	charges []*shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	total := decimal.Zero
	for _, charge := range charges {
		if charge == nil {
			continue
		}

		total = total.Add(CalculateAdditionalCharge(charge, baseCharge))
	}

	return total
}

func (c *Calculator) calculateCommercialTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
	sync chargeSyncOptions,
) (decimal.Decimal, decimal.Decimal, *shipment.RatingDetail, error) {
	if sync.detention {
		if err := c.syncDetentionCharge(ctx, entity, control); err != nil {
			return decimal.Zero, decimal.Zero, nil, err
		}
	}

	baseCharge, ratingDetail, err := c.calculateBaseCharge(ctx, entity, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, nil, err
	}

	// The contract is read once and used twice: it prices its own accessorials,
	// and it may point fuel at a different program than the customer default.
	agreement := c.loadAgreement(ctx, entity)

	// Contract accessorials are applied before fuel, because a fuel program
	// taking a percentage of linehaul plus accessorials has to see them first.
	// A locked shipment is left entirely alone.
	if !entity.RateLocked {
		c.syncAgreementAccessorials(ctx, entity, agreement)
	}

	if sync.fuel {
		if err = c.syncFuelSurcharge(ctx, entity, baseCharge, agreement); err != nil {
			return decimal.Zero, decimal.Zero, nil, err
		}
	}

	return baseCharge, CalculateAdditionalCharges(
		entity.AdditionalCharges,
		baseCharge,
	), ratingDetail, nil
}

func CalculateAdditionalCharge(
	charge *shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	if charge == nil {
		return decimal.Zero
	}

	switch charge.Method {
	case accessorialcharge.MethodFlat:
		unit := max(charge.Unit, 1)
		return charge.Amount.Mul(decimal.NewFromInt32(int32(unit)))
	case accessorialcharge.MethodPerUnit:
		if charge.Unit < 1 {
			return decimal.Zero
		}
		return charge.Amount.Mul(decimal.NewFromInt32(int32(charge.Unit)))
	case accessorialcharge.MethodPercentage:
		return baseCharge.Mul(charge.Amount.Div(decimal.NewFromInt(100)))
	default:
		return decimal.Zero
	}
}

// calculateBaseCharge produces the linehaul.
//
// There are three ways to arrive at one, in strict order of authority. A rate
// somebody set by hand wins outright — re-rating is triggered by a stop edit,
// an assignment, a fuel price job, and any of those silently replacing a
// negotiated spot rate is exactly what makes people switch auto-rating off. A
// locked shipment keeps what it already had, because the customer has seen it.
// Otherwise the rate engine resolves the contract covering the lane, falling
// back to the shipment's own formula template when none does.
func (c *Calculator) calculateBaseCharge(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (decimal.Decimal, *shipment.RatingDetail, error) {
	if entity.RateLocked {
		return c.lockedCharge(entity), entity.RatingDetail, nil
	}

	rated, err := c.rateEngine.RateShipment(ctx, &services.RateShipmentRequest{
		Shipment:  entity,
		Purpose:   ratequote.PurposeRating,
		PartyType: rateagreement.PartyTypeCustomer,
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: userID,
		},
		BillingControl: c.billingControl(ctx, entity),
		Persist:        true,
		UserID:         userID,
	})
	if err != nil {
		return decimal.Zero, nil, err
	}

	c.applyQuote(entity, rated)

	if entity.HasRateOverride() {
		return c.overriddenCharge(entity, rated), entity.RatingDetail, nil
	}

	return rated.Amount, entity.RatingDetail, nil
}

// lockedCharge keeps an invoiced shipment exactly as it was. Nothing is
// recomputed and no quote is written: the numbers have already been billed.
func (c *Calculator) lockedCharge(entity *shipment.Shipment) decimal.Decimal {
	if entity.FreightChargeAmount.Valid {
		return entity.FreightChargeAmount.Decimal
	}

	return decimal.Zero
}

// overriddenCharge returns the hand-set linehaul, and records on the quote what
// the contract would have charged instead.
//
// That difference is the rate leakage report. Storing it on the row that caused
// it, rather than recomputing it later, is the only way it stays true after the
// contract has been amended.
func (c *Calculator) overriddenCharge(
	entity *shipment.Shipment,
	rated *services.RatedShipment,
) decimal.Decimal {
	override := entity.RateOverrideAmount.Decimal

	if rated.Quote != nil {
		rated.Quote.Outcome = ratequote.OutcomeManualOverride
		rated.Quote.OverrideReason = entity.RateOverrideReason
		rated.Quote.ForegoneAmount = decimal.NewNullDecimal(rated.Amount.Sub(override))
		rated.Quote.LinehaulAmount = override
		rated.Quote.TotalAmount = override
		rated.Quote.BillingAmount = override
	}

	if entity.RatingDetail != nil {
		result, _ := override.Float64()
		entity.RatingDetail.Result = result
		entity.RatingDetail.Source = string(ratequote.OutcomeManualOverride)
		entity.RatingDetail.Explanation = overrideExplanation(entity)
	}

	return override
}

func overrideExplanation(entity *shipment.Shipment) string {
	if entity.RateOverrideReason == "" {
		return "Rate set by hand"
	}

	return "Rate set by hand: " + entity.RateOverrideReason
}

// applyQuote stamps the rating decision back onto the shipment, so the record
// of what priced it travels with the shipment rather than only in the quote.
func (c *Calculator) applyQuote(
	entity *shipment.Shipment,
	rated *services.RatedShipment,
) {
	entity.RateAgreementID = rated.AgreementID
	entity.RateAgreementRuleID = rated.RuleID

	if rated.Quote != nil && !rated.Quote.ID.IsNil() {
		quoteID := rated.Quote.ID
		entity.RateQuoteID = &quoteID
	}

	// A rule that delegates to a formula keeps the shipment's template field
	// meaningful, so the existing billing screen goes on showing which formula
	// produced the number.
	if rated.FormulaTemplateID != nil && !rated.FormulaTemplateID.IsNil() {
		entity.FormulaTemplateID = *rated.FormulaTemplateID
	}

	entity.RatingDetail = ratingDetailFromQuote(rated)
}

func ratingDetailFromQuote(rated *services.RatedShipment) *shipment.RatingDetail {
	result, _ := rated.Amount.Float64()

	detail := &shipment.RatingDetail{
		Result: result,
		Source: string(rated.Outcome),
	}

	quote := rated.Quote
	if quote == nil {
		return detail
	}

	detail.RatedAt = quote.RatedAt
	detail.Explanation = quote.Explanation()
	if !quote.ID.IsNil() {
		detail.RateQuoteID = quote.ID.String()
	}
	if quote.RateAgreementID != nil {
		detail.AgreementID = quote.RateAgreementID.String()
	}
	if quote.RateAgreementRuleID != nil {
		detail.RuleID = quote.RateAgreementRuleID.String()
	}
	if quote.FormulaTemplateID != nil {
		detail.FormulaTemplateID = quote.FormulaTemplateID.String()
	}

	if trace := quote.Trace; trace != nil {
		detail.Breakdown = breakdownFromTrace(trace)

		if winner := trace.Winner(); winner != nil {
			detail.AgreementName = winner.AgreementName
			detail.RuleLabel = winner.RuleLabel
		}
	}

	return detail
}

// breakdownFromTrace renders the trace's components as the breakdown lines the
// existing billing screen already knows how to display, so the rating panel
// keeps working unchanged while showing far more than it used to.
func breakdownFromTrace(trace *ratetypes.Trace) []shipment.RatingBreakdownItem {
	if len(trace.Components) == 0 {
		return nil
	}

	items := make([]shipment.RatingBreakdownItem, 0, len(trace.Components))
	for _, component := range trace.Components {
		amount, _ := component.Amount.Float64()
		items = append(items, shipment.RatingBreakdownItem{
			Name:   component.Kind.String(),
			Label:  component.Label,
			Amount: amount,
		})
	}

	return items
}

// billingControl carries the organization's decision about what to do when no
// agreement covers a lane. A failure to load it is not worth failing a save
// over: the default it falls back to is the behaviour that existed before rate
// agreements, which is the safe answer.
func (c *Calculator) billingControl(
	ctx context.Context,
	entity *shipment.Shipment,
) *tenant.BillingControl {
	if c.billingControlRepo == nil {
		return nil
	}

	control, err := c.billingControlRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil
	}

	return control
}

func ratingDate(entity *shipment.Shipment, now func() int64) int64 {
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		return *entity.ActualShipDate
	}
	if entity.CreatedAt > 0 {
		return entity.CreatedAt
	}
	return now()
}

func (c *Calculator) syncDetentionCharge(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
) error {
	if entity == nil || control == nil {
		return nil
	}

	if control.UseDetentionPolicyEngine {
		return c.syncDetentionFromPolicyEngine(ctx, entity)
	}

	if !control.TrackDetentionTime ||
		!control.AutoGenerateDetentionCharges ||
		control.DetentionChargeID == nil ||
		control.DetentionChargeID.IsNil() {
		return nil
	}

	accessorial, err := c.accessorialRepo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID: *control.DetentionChargeID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}

	thresholdMinutes := shipmentstate.DefaultDelayThresholdMinutes
	if control.DetentionThreshold != nil {
		thresholdMinutes = shipmentstate.ResolveDelayThresholdMinutes(*control.DetentionThreshold)
	}

	totalExcessSeconds, detainedStopCount := detentionExposure(entity, c.now(), thresholdMinutes)
	if totalExcessSeconds <= 0 || detainedStopCount == 0 {
		removeGeneratedDetentionCharges(entity, accessorial.ID)
		return nil
	}

	unit := max(detentionUnits(accessorial.RateUnit, totalExcessSeconds, detainedStopCount), 1)

	ensureGeneratedDetentionCharge(entity, accessorial, unit)
	return nil
}

func detentionExposure(
	entity *shipment.Shipment,
	currentTime int64,
	thresholdMinutes int16,
) (int64, int16) {
	thresholdSeconds := int64(shipmentstate.ResolveDelayThresholdMinutes(thresholdMinutes)) * 60
	var totalExcessSeconds int64
	var detainedStopCount int16

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipment.StopStatusCanceled ||
				stop.ActualArrival == nil {
				continue
			}

			endTime := currentTime
			if stop.ActualDeparture != nil && *stop.ActualDeparture > 0 {
				endTime = *stop.ActualDeparture
			}

			if endTime <= *stop.ActualArrival {
				continue
			}

			excessSeconds := endTime - *stop.ActualArrival - thresholdSeconds
			if excessSeconds <= 0 {
				continue
			}

			totalExcessSeconds += excessSeconds
			detainedStopCount++
		}
	}

	return totalExcessSeconds, detainedStopCount
}

func detentionUnits(
	rateUnit accessorialcharge.RateUnit,
	totalExcessSeconds int64,
	detainedStopCount int16,
) int16 {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch rateUnit {
	case accessorialcharge.RateUnitHour:
		return int16(math.Ceil(float64(totalExcessSeconds) / 3600))
	case accessorialcharge.RateUnitDay:
		return int16(math.Ceil(float64(totalExcessSeconds) / 86400))
	case accessorialcharge.RateUnitStop:
		return detainedStopCount
	default:
		return 1
	}
}

func (c *Calculator) syncFuelSurcharge(
	ctx context.Context,
	entity *shipment.Shipment,
	baseCharge decimal.Decimal,
	agreement *rateagreement.RateAgreement,
) error {
	if entity == nil || c.fuelSurcharge == nil {
		return nil
	}

	if entity.FuelSurchargeLocked {
		return nil
	}

	resolved, err := c.fuelSurcharge.ResolveShipmentCharge(
		ctx,
		&services.ResolveShipmentChargeRequest{
			Shipment:         entity,
			Linehaul:         baseCharge,
			AccessorialTotal: nonFuelSurchargeChargeTotal(entity, baseCharge),
			Override:         fuelOverride(agreement),
		},
	)
	if err != nil {
		return err
	}

	if resolved == nil || resolved.Amount.LessThanOrEqual(decimal.Zero) {
		removeGeneratedFuelSurchargeCharges(entity)
		return nil
	}

	ensureGeneratedFuelSurchargeCharge(entity, resolved)
	return nil
}

func nonFuelSurchargeChargeTotal(
	entity *shipment.Shipment,
	baseCharge decimal.Decimal,
) decimal.Decimal {
	total := decimal.Zero
	for _, charge := range entity.AdditionalCharges {
		if charge == nil || (charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil) {
			continue
		}
		total = total.Add(CalculateAdditionalCharge(charge, baseCharge))
	}
	return total
}

func removeGeneratedFuelSurchargeCharges(entity *shipment.Shipment) {
	filtered := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil {
			continue
		}
		filtered = append(filtered, charge)
	}
	entity.AdditionalCharges = filtered
}

func ensureGeneratedFuelSurchargeCharge(
	entity *shipment.Shipment,
	resolved *services.ResolvedFuelSurcharge,
) {
	var generated *shipment.AdditionalCharge
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if !charge.IsSystemGenerated || charge.FuelSurchargeProgramID == nil {
			filtered = append(filtered, charge)
			continue
		}

		if generated == nil {
			generated = charge
		}
	}

	if generated == nil {
		generated = &shipment.AdditionalCharge{
			OrganizationID:    entity.OrganizationID,
			BusinessUnitID:    entity.BusinessUnitID,
			ShipmentID:        entity.ID,
			IsSystemGenerated: true,
		}
	}

	programID := resolved.ProgramID
	generated.IsSystemGenerated = true
	generated.AccessorialChargeID = resolved.AccessorialChargeID
	generated.Method = accessorialcharge.MethodFlat
	generated.Amount = resolved.Amount
	generated.Unit = 1
	generated.FuelSurchargeProgramID = &programID
	generated.FuelSurchargeDetail = resolved.Detail

	filtered = append(filtered, generated)
	entity.AdditionalCharges = filtered
}

func removeGeneratedDetentionCharges(entity *shipment.Shipment, detentionChargeID pulid.ID) {
	filtered := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.AccessorialChargeID == detentionChargeID && charge.IsSystemGenerated {
			continue
		}
		filtered = append(filtered, charge)
	}
	entity.AdditionalCharges = filtered
}

func ensureGeneratedDetentionCharge(
	entity *shipment.Shipment,
	accessorial *accessorialcharge.AccessorialCharge,
	unit int16,
) {
	var generated *shipment.AdditionalCharge
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if charge.AccessorialChargeID != accessorial.ID || !charge.IsSystemGenerated {
			filtered = append(filtered, charge)
			continue
		}

		if generated == nil {
			generated = charge
		}
	}

	if generated == nil {
		generated = &shipment.AdditionalCharge{
			OrganizationID:      entity.OrganizationID,
			BusinessUnitID:      entity.BusinessUnitID,
			ShipmentID:          entity.ID,
			AccessorialChargeID: accessorial.ID,
			IsSystemGenerated:   true,
			Method:              accessorial.Method,
			Amount:              accessorial.Amount,
		}
	}

	generated.IsSystemGenerated = true
	generated.Unit = unit

	filtered = append(filtered, generated)
	entity.AdditionalCharges = filtered
}

// syncDetentionFromPolicyEngine rebuilds detention charges from resolved policy
// occurrences. Each billable occurrence contributes exactly one system charge
// carrying the amount the engine already computed, so the invoice line and the
// stored calculation trace can never disagree.
func (c *Calculator) syncDetentionFromPolicyEngine(
	ctx context.Context,
	entity *shipment.Shipment,
) error {
	if c.detentionEngine == nil {
		return nil
	}

	result, err := c.detentionEngine.SyncShipment(ctx, entity)
	if err != nil {
		return err
	}

	if result == nil {
		reconcileDetentionOccurrenceCharges(entity, nil)
		return nil
	}

	reconcileDetentionOccurrenceCharges(entity, result.Occurrences)

	return nil
}

// reconcileDetentionOccurrenceCharges rewrites the shipment's detention charges
// so exactly one charge survives per billable occurrence. Existing rows are
// reused rather than replaced: a charge that keeps its identity keeps its audit
// trail, and the occurrence it points at stays resolvable after the save.
func reconcileDetentionOccurrenceCharges(
	entity *shipment.Shipment,
	occurrences []*detention.DetentionOccurrence,
) {
	existing := make(map[pulid.ID]*shipment.AdditionalCharge, len(occurrences))
	filtered := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}

		if !charge.IsSystemGenerated || charge.DetentionOccurrenceID == nil {
			filtered = append(filtered, charge)
			continue
		}

		if _, ok := existing[*charge.DetentionOccurrenceID]; !ok {
			existing[*charge.DetentionOccurrenceID] = charge
		}
	}

	for _, occurrence := range occurrences {
		if !detentionOccurrenceIsChargeable(occurrence) {
			continue
		}

		filtered = append(filtered, detentionOccurrenceCharge(
			entity,
			occurrence,
			existing[occurrence.ID],
		))
	}

	entity.AdditionalCharges = filtered
}

func detentionOccurrenceIsChargeable(occurrence *detention.DetentionOccurrence) bool {
	return occurrence != nil &&
		occurrence.PolicySnapshot != nil &&
		occurrence.Status.IsBillable() &&
		occurrence.BillableAmount.GreaterThan(decimal.Zero)
}

func detentionOccurrenceCharge(
	entity *shipment.Shipment,
	occurrence *detention.DetentionOccurrence,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	occurrenceID := occurrence.ID

	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.AccessorialChargeID = occurrence.PolicySnapshot.AccessorialChargeID
	charge.Method = accessorialcharge.MethodFlat
	charge.Amount = occurrence.BillableAmount
	charge.Unit = 1
	charge.DetentionOccurrenceID = &occurrenceID

	return charge
}

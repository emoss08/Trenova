package rateengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

var (
	// ErrUnsupportedBasis guards the switch over rating bases. It can only fire
	// if a basis is added to the enum without being priced here.
	ErrUnsupportedBasis = errors.New("rating basis is not supported")

	// ErrMissingRate is returned when a rule reaches pricing without the input
	// its basis needs. Validation rejects this at write time, so it means a row
	// was written around the domain.
	ErrMissingRate = errors.New("rule has no rate to price with")

	// ErrPercentWithoutBasis is returned when a percentage rule has nothing to
	// take a percentage of.
	ErrPercentWithoutBasis = errors.New("percentage rule has no basis amount")
)

var (
	oneHundred        = decimal.NewFromInt(100)
	poundsPerCwtUnits = decimal.NewFromInt(100)
)

// priceResult is the linehaul a rule produced, before fuel and accessorials.
type priceResult struct {
	Amount   decimal.Decimal
	Currency string
}

// price computes what a rule charges, recording each step on the trace as it
// goes.
//
// The order is the order a contract is read in, and the trace records it that
// way because the order is itself contentious: a discount taken before a
// minimum charge produces a different number than one taken after, and which a
// tariff means is the sort of thing disputes are made of. Here it is: base
// charge, discount, absolute minimum, deficit distance, then the rule's own
// minimum and maximum.
func (s *Service) price(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (priceResult, error) {
	agreement := rule.Agreement
	currency := rule.EffectiveCurrency(agreement)

	base, err := s.baseCharge(ctx, rateCtx, rule, trace)
	if err != nil {
		return priceResult{}, err
	}

	amount := base
	amount = applyDiscount(rule, amount, trace)
	amount = applyAbsoluteMinimum(rule, amount, trace)
	amount = s.applyDeficitDistance(ctx, rateCtx, rule, amount, trace)
	amount = applyGuardrails(rule, agreement, amount, trace)
	amount = applyRounding(rule, agreement, amount, trace)

	return priceResult{Amount: amount, Currency: currency}, nil
}

//nolint:cyclop // one arm per rating basis; splitting it would only hide the shape
func (s *Service) baseCharge(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	switch rule.RatingBasis {
	case rateagreement.RatingBasisFlat:
		return s.flatCharge(rateCtx, rule, trace)
	case rateagreement.RatingBasisPerMile:
		return unitCharge(rule, rateCtx.Distance, "mi", trace)
	case rateagreement.RatingBasisPerCwt:
		return s.hundredweightCharge(rateCtx, rule, trace)
	case rateagreement.RatingBasisPerPiece:
		return unitCharge(rule, decimal.NewFromInt(rateCtx.Pieces), "pieces", trace)
	case rateagreement.RatingBasisPerStop:
		return unitCharge(rule, decimal.NewFromInt(int64(rateCtx.Stops)), "stops", trace)
	case rateagreement.RatingBasisPerPallet:
		return unitCharge(rule, decimal.NewFromInt(rateCtx.Pallets), "pallets", trace)
	case rateagreement.RatingBasisPerLinearFoot:
		return unitCharge(rule, rateCtx.LinearFeet, "linear ft", trace)
	case rateagreement.RatingBasisPerHour:
		return unitCharge(rule, rateCtx.Hours, "hrs", trace)
	case rateagreement.RatingBasisPercent:
		return percentCharge(rateCtx, rule, trace)
	case rateagreement.RatingBasisMatrix:
		return s.matrixCharge(ctx, rateCtx, rule, trace)
	case rateagreement.RatingBasisFormula:
		return s.formulaCharge(ctx, rateCtx, rule, trace)
	default:
		return decimal.Zero, fmt.Errorf("%w: %s", ErrUnsupportedBasis, rule.RatingBasis)
	}
}

// flatCharge is a fixed amount, unless the rule bands its flat rates by weight,
// which is how "under 5,000 lb costs this, over costs that" is written.
func (s *Service) flatCharge(
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if len(rule.Breaks) > 0 {
		result, err := rateagreement.ApplyDeficitRating(
			rule.Breaks,
			rateCtx.Weight,
			rule.AllowDeficitRating,
		)
		if err != nil {
			return decimal.Zero, err
		}

		recordBreak(rule, result, trace, "flat")

		return result.UsedBreak.Rate, nil
	}

	if !rule.Rate.Valid {
		return decimal.Zero, ErrMissingRate
	}

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      "Flat rate",
		Basis:      "flat",
		Amount:     rule.Rate.Decimal,
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	return rule.Rate.Decimal, nil
}

func unitCharge(
	rule *rateagreement.RateAgreementRule,
	quantity decimal.Decimal,
	unit string,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if !rule.Rate.Valid {
		return decimal.Zero, ErrMissingRate
	}

	amount := rule.Rate.Decimal.Mul(quantity)

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      "Linehaul",
		Basis:      quantity.String() + " " + unit + " @ " + rule.Rate.Decimal.String(),
		Quantity:   decimal.NewNullDecimal(quantity),
		Rate:       rule.Rate,
		Amount:     amount,
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	return amount, nil
}

// hundredweightCharge prices by weight, through the rule's bands when it has
// them and at its single rate when it does not.
//
// Class tariffs get cheaper per hundredweight as freight gets heavier, so the
// banded path also applies deficit rating: a shipment near the top of a band
// can be cheaper declared at the bottom of the next one up, and the carrier
// lets the shipper pay the lower of the two.
func (s *Service) hundredweightCharge(
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if len(rule.Breaks) == 0 {
		if !rule.Rate.Valid {
			return decimal.Zero, ErrMissingRate
		}

		hundredweights := rateCtx.Weight.Div(poundsPerCwtUnits)
		amount := rule.Rate.Decimal.Mul(hundredweights)

		trace.AddComponent(ratetypes.Component{
			Kind:  ratetypes.ComponentKindLinehaul,
			Label: "Linehaul",
			Basis: rateCtx.Weight.String() + " lb (" + hundredweights.String() +
				" cwt) @ " + rule.Rate.Decimal.String(),
			Quantity:   decimal.NewNullDecimal(hundredweights),
			Rate:       rule.Rate,
			Amount:     amount,
			Source:     ratetypes.ComponentSourceAgreementRule,
			SourceID:   rule.ID.String(),
			SourceName: rule.Label,
		})

		return amount, nil
	}

	result, err := rateagreement.ApplyDeficitRating(
		rule.Breaks,
		rateCtx.Weight,
		rule.AllowDeficitRating,
	)
	if err != nil {
		return decimal.Zero, err
	}

	recordBreak(rule, result, trace, "cwt")

	return result.Charge, nil
}

// recordBreak writes the band that priced the shipment, and separately records
// a bump when one happened.
//
// The bump gets its own line because it is the first thing a shipper questions
// on an LTL invoice — the weight billed is not the weight shipped — and the
// answer is that it cost them less.
func recordBreak(
	rule *rateagreement.RateAgreementRule,
	result rateagreement.BreakResult,
	trace *ratetypes.Trace,
	unit string,
) {
	detail := map[string]any{
		"fromWeight":   result.UsedBreak.FromWeight.String(),
		"actualWeight": result.ActualWeight.String(),
		"ratedWeight":  result.RatedWeight.String(),
	}
	if result.UsedBreak.ToWeight.Valid {
		detail["toWeight"] = result.UsedBreak.ToWeight.Decimal.String()
	}

	trace.AddComponent(ratetypes.Component{
		Kind:  ratetypes.ComponentKindWeightBreak,
		Label: "Weight break " + result.UsedBreak.DisplayLabel(),
		Basis: result.RatedWeight.String() + " lb @ " +
			result.UsedBreak.Rate.String() + "/" + unit,
		Quantity:   decimal.NewNullDecimal(result.RatedWeight),
		Rate:       decimal.NewNullDecimal(result.UsedBreak.Rate),
		Amount:     result.Charge,
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
		Detail:     detail,
	})

	if !result.Bumped {
		return
	}

	trace.Warn(
		"Rated as though the shipment weighed " + result.RatedWeight.String() +
			" lb, which costs less than its actual " + result.ActualWeight.String() + " lb",
	)
}

// percentCharge takes a share of another amount, which is how buy side rates
// written as a percentage of what the customer pays are expressed.
func percentCharge(
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if !rule.Rate.Valid {
		return decimal.Zero, ErrMissingRate
	}

	var basis decimal.Decimal

	switch rule.PercentBasis {
	case rateagreement.PercentBasisSellRate:
		if !rateCtx.SellTotal.Valid {
			return decimal.Zero, ErrPercentWithoutBasis
		}
		basis = rateCtx.SellTotal.Decimal
	case rateagreement.PercentBasisLinehaul,
		rateagreement.PercentBasisLinehaulPlusAccessorials:
		// Both of these describe a share of the sell side's own linehaul, which
		// on the buy side is the only amount already known at this point. The
		// distinction between them is applied when accessorials are summed.
		if !rateCtx.SellTotal.Valid {
			return decimal.Zero, ErrPercentWithoutBasis
		}
		basis = rateCtx.SellTotal.Decimal
	default:
		return decimal.Zero, ErrPercentWithoutBasis
	}

	amount := basis.Mul(rule.Rate.Decimal).Div(oneHundred)

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      "Linehaul",
		Basis:      rule.Rate.Decimal.String() + "% of " + basis.String(),
		Quantity:   decimal.NewNullDecimal(basis),
		Rate:       rule.Rate,
		Amount:     amount,
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	return amount, nil
}

// formulaCharge hands the rule to the existing formula engine.
//
// This is the escape hatch that keeps the structured bases from having to cover
// everything. It also means every formula template an organization has already
// written keeps working — the difference is only that the contract now chooses
// which one applies, instead of somebody picking it on the shipment.
func (s *Service) formulaCharge(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if rule.FormulaTemplateID == nil || rule.FormulaTemplateID.IsNil() {
		return decimal.Zero, ErrMissingRate
	}

	resp, err := s.formula.Calculate(ctx, &formulatemplatetypes.CalculateRequest{
		TemplateID: *rule.FormulaTemplateID,
		Entity:     rateCtx.Entity,
		TenantInfo: rateCtx.TenantInfo,
		RatingDate: rateCtx.AsOf,
	})
	if err != nil {
		return decimal.Zero, err
	}

	detail := map[string]any{"expression": resp.Expression}
	if resp.VersionNumber > 0 {
		detail["versionNumber"] = resp.VersionNumber
	}

	// The template's own named breakdown lines carry through onto the
	// component, so a formula priced rate is as readable in the trace as a
	// structured one rather than arriving as a single unexplained number.
	if len(resp.Breakdown) > 0 {
		lines := make(map[string]string, len(resp.Breakdown))
		for _, item := range resp.Breakdown {
			label := item.Label
			if label == "" {
				label = item.Name
			}
			if item.Error != "" {
				lines[label] = item.Error
				trace.Warn("Breakdown line " + label + " failed: " + item.Error)
				continue
			}
			lines[label] = item.Amount.String()
		}
		detail["breakdown"] = lines
	}

	if resp.Guardrail != nil && resp.Guardrail.Applied {
		trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
			Kind:    guardrailKindForBound(resp.Guardrail.Bound),
			Applied: true,
			Raw:     resp.Guardrail.RawAmount,
			Result:  resp.Amount,
		})
	}

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      "Linehaul",
		Basis:      resp.FormulaTemplateName,
		Amount:     resp.Amount,
		Source:     ratetypes.ComponentSourceFormulaTemplate,
		SourceID:   resp.FormulaTemplateID,
		SourceName: resp.FormulaTemplateName,
		Detail:     detail,
	})

	return resp.Amount, nil
}

// guardrailKindForBound translates the formula engine's own bound naming into
// the trace vocabulary, so a clamp reads the same however it was applied.
func guardrailKindForBound(bound string) ratetypes.ComponentKind {
	if bound == "max" {
		return ratetypes.ComponentKindMaximumCharge
	}

	return ratetypes.ComponentKindMinimumCharge
}

func applyDiscount(
	rule *rateagreement.RateAgreementRule,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	if !rule.DiscountPercent.Valid || rule.DiscountPercent.Decimal.IsZero() {
		return amount
	}

	discount := amount.Mul(rule.DiscountPercent.Decimal).Div(oneHundred)

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindDiscount,
		Label:      "Discount",
		Basis:      rule.DiscountPercent.Decimal.String() + "% off " + amount.String(),
		Rate:       rule.DiscountPercent,
		Amount:     discount.Neg(),
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	return amount.Sub(discount)
}

// applyAbsoluteMinimum is the LTL floor that a discount cannot take a shipment
// below. It sits between the discount and the rule's own minimum because that
// is the order a class tariff states them in.
func applyAbsoluteMinimum(
	rule *rateagreement.RateAgreementRule,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	if !rule.AbsoluteMinCharge.Valid || !amount.LessThan(rule.AbsoluteMinCharge.Decimal) {
		return amount
	}

	trace.AddComponent(ratetypes.Component{
		Kind:       ratetypes.ComponentKindAbsoluteMinCharge,
		Label:      "Absolute minimum charge",
		Basis:      "raised from " + amount.String(),
		Amount:     rule.AbsoluteMinCharge.Decimal.Sub(amount),
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
		Kind:    ratetypes.ComponentKindAbsoluteMinCharge,
		Applied: true,
		Bound:   rule.AbsoluteMinCharge.Decimal,
		Raw:     amount,
		Result:  rule.AbsoluteMinCharge.Decimal,
	})

	return rule.AbsoluteMinCharge.Decimal
}

// applyDeficitDistance bills a short haul as though it ran the minimum mileage
// the contract guarantees, which is how a carrier is kept whole on a lane too
// short to be worth dispatching.
//
// It only means anything to a per-mile rate; on any other basis the miles are
// not what is being multiplied.
func (s *Service) applyDeficitDistance(
	_ context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	if rule.RatingBasis != rateagreement.RatingBasisPerMile ||
		!rule.MinBillableDistance.Valid || !rule.Rate.Valid {
		return amount
	}

	if !rateCtx.Distance.LessThan(rule.MinBillableDistance.Decimal) {
		return amount
	}

	billed := rule.Rate.Decimal.Mul(rule.MinBillableDistance.Decimal)

	trace.AddComponent(ratetypes.Component{
		Kind:  ratetypes.ComponentKindDeficitDistance,
		Label: "Minimum billable distance",
		Basis: "billed at " + rule.MinBillableDistance.Decimal.String() +
			" mi rather than " + rateCtx.Distance.String(),
		Quantity:   rule.MinBillableDistance,
		Rate:       rule.Rate,
		Amount:     billed.Sub(amount),
		Source:     ratetypes.ComponentSourceAgreementRule,
		SourceID:   rule.ID.String(),
		SourceName: rule.Label,
	})

	return billed
}

// applyGuardrails clamps to the rule's own bounds, falling back to the
// agreement's defaults where the rule states none.
//
// Both are applied in the agreement's currency, before any conversion. A
// contractual minimum charge converted first would drift with the exchange
// rate, which is a breach of the contract rather than a rounding difference.
func applyGuardrails(
	rule *rateagreement.RateAgreementRule,
	agreement *rateagreement.RateAgreement,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	minimum := rule.MinCharge
	if !minimum.Valid && agreement != nil {
		minimum = agreement.DefaultMinCharge
	}

	maximum := rule.MaxCharge
	if !maximum.Valid && agreement != nil {
		maximum = agreement.DefaultMaxCharge
	}

	if minimum.Valid && amount.LessThan(minimum.Decimal) {
		trace.AddComponent(ratetypes.Component{
			Kind:       ratetypes.ComponentKindMinimumCharge,
			Label:      "Minimum charge",
			Basis:      "raised from " + amount.String(),
			Amount:     minimum.Decimal.Sub(amount),
			Source:     ratetypes.ComponentSourceAgreementRule,
			SourceID:   rule.ID.String(),
			SourceName: rule.Label,
		})
		trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
			Kind:    ratetypes.ComponentKindMinimumCharge,
			Applied: true,
			Bound:   minimum.Decimal,
			Raw:     amount,
			Result:  minimum.Decimal,
		})

		return minimum.Decimal
	}

	if maximum.Valid && amount.GreaterThan(maximum.Decimal) {
		trace.AddComponent(ratetypes.Component{
			Kind:       ratetypes.ComponentKindMaximumCharge,
			Label:      "Maximum charge",
			Basis:      "reduced from " + amount.String(),
			Amount:     maximum.Decimal.Sub(amount),
			Source:     ratetypes.ComponentSourceAgreementRule,
			SourceID:   rule.ID.String(),
			SourceName: rule.Label,
		})
		trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
			Kind:    ratetypes.ComponentKindMaximumCharge,
			Applied: true,
			Bound:   maximum.Decimal,
			Raw:     amount,
			Result:  maximum.Decimal,
		})

		return maximum.Decimal
	}

	return amount
}

func applyRounding(
	rule *rateagreement.RateAgreementRule,
	agreement *rateagreement.RateAgreement,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	precision := int32(2)
	if agreement != nil {
		precision = int32(agreement.RoundingPrecision)
	}

	rounded := rule.EffectiveRoundingMode(agreement).Round(amount, precision)
	if rounded.Equal(amount) {
		return amount
	}

	trace.AddComponent(ratetypes.Component{
		Kind:   ratetypes.ComponentKindRounding,
		Label:  "Rounding",
		Basis:  rule.EffectiveRoundingMode(agreement).String(),
		Amount: rounded.Sub(amount),
		Source: ratetypes.ComponentSourceAgreementRule,
	})

	return rounded
}

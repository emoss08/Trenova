package rateengine

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

// ErrMissingRate is returned when a rule reaches pricing with neither a
// formula template nor a rate matrix. Validation rejects this at write time,
// so it means a row was written around the domain.
var ErrMissingRate = errors.New("rule has no rating method to price with")

var oneHundred = decimal.NewFromInt(100)

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

// baseCharge dispatches to the rule's pricing method: a rate matrix when the
// rule names one, and its formula template otherwise. Every rate in the system
// is ultimately produced by a formula template — a matrix only decides which
// number the matrix's own template prices with.
func (s *Service) baseCharge(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if rule.HasRateMatrix() {
		return s.matrixCharge(ctx, rateCtx, rule, trace)
	}

	return s.formulaCharge(ctx, rateCtx, rule, trace)
}

// formulaCharge prices the rule through its formula template.
//
// The rule's own rate binds into the template as its base rate, so "two
// dollars fifteen a mile" is a Per Mile template plus a rate on the lane —
// the analyst types a number, never an expression. A rule without a rate lets
// the shipment's own base rate flow through, which is how a template that
// reads no rate at all (a share of existing charges, say) is used.
//
// Banded rules resolve their base rate through the weight breaks first, with
// deficit rating applied: a shipment near the top of a band can be cheaper
// declared at the bottom of the next one up, and the carrier lets the shipper
// pay the lower of the two. The template then prices at the band's rate and
// the rated weight.
func (s *Service) formulaCharge(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	if !rule.HasFormulaTemplate() {
		return decimal.Zero, ErrMissingRate
	}

	overrides := make(map[string]any, 3)
	detail := make(map[string]any, 8)
	boundRate := rule.Rate

	var breakMinCharge decimal.NullDecimal

	if len(rule.Breaks) > 0 {
		result, err := rateagreement.ApplyDeficitRating(
			rule.Breaks,
			rateCtx.Weight,
			rule.AllowDeficitRating,
		)
		if err != nil {
			return decimal.Zero, err
		}

		boundRate = decimal.NewNullDecimal(result.UsedBreak.Rate)
		breakMinCharge = result.UsedBreak.MinCharge
		overrides["totalWeight"] = result.RatedWeight.InexactFloat64()
		recordBreakSelection(result, detail, trace)
	}

	if boundRate.Valid {
		overrides["baseRate"] = boundRate.Decimal.InexactFloat64()
	}

	// SellTotal is only present on the buy side; binding it is what lets a
	// carrier agreement be written as a share of what the customer pays.
	if rateCtx.SellTotal.Valid {
		overrides["sellTotal"] = rateCtx.SellTotal.Decimal.InexactFloat64()
	}

	resp, err := s.formula.Calculate(ctx, &formulatemplatetypes.CalculateRequest{
		TemplateID: *rule.FormulaTemplateID,
		Entity:     rateCtx.Entity,
		Overrides:  overrides,
		TenantInfo: rateCtx.TenantInfo,
		RatingDate: rateCtx.AsOf,
	})
	if err != nil {
		return decimal.Zero, err
	}

	amount := resp.Amount
	if breakMinCharge.Valid && amount.LessThan(breakMinCharge.Decimal) {
		trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
			Kind:    ratetypes.ComponentKindMinimumCharge,
			Applied: true,
			Bound:   breakMinCharge.Decimal,
			Raw:     amount,
			Result:  breakMinCharge.Decimal,
		})
		amount = breakMinCharge.Decimal
	}

	recordFormulaResponse(resp, detail, trace)

	basis := resp.FormulaTemplateName
	if boundRate.Valid {
		basis += " @ " + boundRate.Decimal.String()
	}

	trace.AddComponent(&ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      linehaulLabel,
		Basis:      basis,
		Rate:       boundRate,
		Amount:     amount,
		Source:     ratetypes.ComponentSourceFormulaTemplate,
		SourceID:   resp.FormulaTemplateID,
		SourceName: resp.FormulaTemplateName,
		Detail:     detail,
	})

	return amount, nil
}

// recordBreakSelection writes which band priced the shipment into the trace
// detail, and separately warns when a bump happened.
//
// The bump gets its own line because it is the first thing a shipper questions
// on an LTL invoice — the weight billed is not the weight shipped — and the
// answer is that it cost them less.
func recordBreakSelection(
	result rateagreement.BreakResult,
	detail map[string]any,
	trace *ratetypes.Trace,
) {
	breakDetail := map[string]any{
		"label":        result.UsedBreak.DisplayLabel(),
		"rate":         result.UsedBreak.Rate.String(),
		"fromWeight":   result.UsedBreak.FromWeight.String(),
		"actualWeight": result.ActualWeight.String(),
		"ratedWeight":  result.RatedWeight.String(),
	}
	if result.UsedBreak.ToWeight.Valid {
		breakDetail["toWeight"] = result.UsedBreak.ToWeight.Decimal.String()
	}
	detail["weightBreak"] = breakDetail

	if !result.Bumped {
		return
	}

	trace.Warn(
		"Rated as though the shipment weighed " + result.RatedWeight.String() +
			" lb, which costs less than its actual " + result.ActualWeight.String() + " lb",
	)
}

// recordFormulaResponse writes what the template computed into the trace
// detail, so a formula priced rate is as readable in the trace as a structured
// one rather than arriving as a single unexplained number.
func recordFormulaResponse(
	resp *formulatemplatetypes.CalculateResponse,
	detail map[string]any,
	trace *ratetypes.Trace,
) {
	detail["expression"] = resp.Expression
	// The template is recorded by id as well as by name, because reproducing a
	// rating — or seating it on a shipment as its own rating method — needs the
	// id, and a matrix-priced rule names its template nowhere else.
	detail["formulaTemplateId"] = resp.FormulaTemplateID
	if resp.VersionNumber > 0 {
		detail["versionNumber"] = resp.VersionNumber
	}
	// The receipt is what lets a person read the number back: each variable
	// with its source, each table row consulted. Timing is dropped because a
	// trace is compared and stored, and wall-clock time is noise in both.
	if resp.Receipt != nil {
		detail["receipt"] = resp.Receipt.WithoutTiming()
	}

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

	trace.AddComponent(&ratetypes.Component{
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

	trace.AddComponent(&ratetypes.Component{
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
// It only means anything on a per-mile lane, which is why it reads the rule's
// own rate: a minimum billable distance is only ever written on a rule whose
// rate is a per-mile number.
func (s *Service) applyDeficitDistance(
	_ context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	amount decimal.Decimal,
	trace *ratetypes.Trace,
) decimal.Decimal {
	if !rule.MinBillableDistance.Valid || !rule.Rate.Valid {
		return amount
	}

	if !rateCtx.Distance.LessThan(rule.MinBillableDistance.Decimal) {
		return amount
	}

	billed := rule.Rate.Decimal.Mul(rule.MinBillableDistance.Decimal)

	trace.AddComponent(&ratetypes.Component{
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
		trace.AddComponent(&ratetypes.Component{
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
		trace.AddComponent(&ratetypes.Component{
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

	trace.AddComponent(&ratetypes.Component{
		Kind:   ratetypes.ComponentKindRounding,
		Label:  "Rounding",
		Basis:  rule.EffectiveRoundingMode(agreement).String(),
		Amount: rounded.Sub(amount),
		Source: ratetypes.ComponentSourceAgreementRule,
	})

	return rounded
}

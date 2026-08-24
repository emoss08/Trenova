package carrierassignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// buySideResult is what a carrier's contract said this move is worth, in the
// shape the assignment request already takes.
type buySideResult struct {
	RateMethod    shipment.CarrierRateMethod
	BaseRate      decimal.Decimal
	FuelSurcharge decimal.Decimal
	Accessorials  []repositories.CarrierAccessorialInput
	RateQuoteID   *pulid.ID
	Margin        ratetypes.MarginVerdict
}

// shouldAutoRate reports whether the contract should price this assignment.
//
// A rate somebody typed is a decision, and auto-rating over the top of it would
// be the system overruling a negotiated number. That is precisely what makes
// people switch auto-rating off, so a typed rate always wins.
func shouldAutoRate(req *repositories.AssignMoveToCarrierRequest) bool {
	return req != nil && req.AutoRate && req.BaseRate.IsZero()
}

// buySideRating turns a rated buy side into the numbers an assignment carries.
//
// The linehaul becomes a flat amount rather than a rate and a distance. The
// engine has already applied minimums, deficit miles and rounding to reach its
// number, and splitting it back into a per-mile rate would let the assignment
// recompute a total that disagrees with its own quote — the exact failure
// contract pricing exists to remove. The quote is where the arithmetic lives,
// and the assignment points at it.
func buySideRating(
	rated *services.RatedShipment,
	agreement *rateagreement.RateAgreement,
	sellTotal decimal.NullDecimal,
) (*buySideResult, error) {
	if rated == nil {
		return nil, errortypes.NewBusinessError(
			"No carrier agreement covers this lane, so the carrier's pay could not be priced",
		)
	}

	if !ratedOnContract(rated) {
		return nil, errortypes.NewBusinessError(
			"No carrier agreement covers this lane, so the carrier's pay could not be priced",
		)
	}

	result := &buySideResult{
		RateMethod:   shipment.CarrierRateMethodFlat,
		BaseRate:     rated.Amount,
		RateQuoteID:  quoteIDOf(rated),
		Accessorials: contractAccessorials(agreement),
	}

	result.Margin = ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      valueOf(sellTotal),
		Buy:       rated.Amount,
		Floor:     floorOf(agreement),
		MaxPayPct: ceilingOf(agreement),
	})

	return result, nil
}

// ratedOnContract reports whether a carrier contract actually put a number on
// this move. The buy side has no formula fallback, so an outcome that produced
// an amount is always a contract's amount.
func ratedOnContract(rated *services.RatedShipment) bool {
	return rated.Outcome.Priced()
}

func quoteIDOf(rated *services.RatedShipment) *pulid.ID {
	if rated.Quote == nil || rated.Quote.ID.IsNil() {
		return nil
	}

	id := rated.Quote.ID

	return &id
}

// contractAccessorials reads the carrier contract's own schedule.
//
// It is the same list the sell side reads, so a rate confirmation and a carrier
// settlement cannot disagree about what the carrier is owed.
//
// Rows whose application depends on a formula are left out. This path has no
// engine to answer one with, and paying a carrier for a condition nobody
// checked is worse than leaving a charge for somebody to add deliberately.
func contractAccessorials(
	agreement *rateagreement.RateAgreement,
) []repositories.CarrierAccessorialInput {
	scoped := agreement.AutoApplyAccessorials(rateagreement.AccessorialFacts{})

	inputs := make([]repositories.CarrierAccessorialInput, 0, len(scoped))

	for _, row := range scoped {
		if row.ApplyCondition != "" {
			continue
		}

		chargeID := row.AccessorialChargeID
		inputs = append(inputs, repositories.CarrierAccessorialInput{
			AccessorialChargeID: &chargeID,
			Description:         accessorialDescription(row),
			Amount:              row.PricedAmount(),
		})
	}

	return inputs
}

func accessorialDescription(row *rateagreement.RateAgreementAccessorial) string {
	if row.AccessorialCharge == nil {
		return ""
	}

	if row.AccessorialCharge.Description != "" {
		return row.AccessorialCharge.Description
	}

	return row.AccessorialCharge.Code
}

func floorOf(agreement *rateagreement.RateAgreement) decimal.NullDecimal {
	if agreement == nil {
		return decimal.NullDecimal{}
	}

	return agreement.MarginFloorPercent
}

func ceilingOf(agreement *rateagreement.RateAgreement) decimal.NullDecimal {
	if agreement == nil {
		return decimal.NullDecimal{}
	}

	return agreement.MaxPayPercentOfSell
}

func valueOf(value decimal.NullDecimal) decimal.Decimal {
	if !value.Valid {
		return decimal.Zero
	}

	return value.Decimal
}

// applyBuySideRating writes the contract's answer onto the request.
//
// Filling the request rather than the assignment is deliberate: everything
// downstream — validation, totals, the rate confirmation — then runs on one
// path whether a person typed the numbers or a contract produced them.
func applyBuySideRating(
	req *repositories.AssignMoveToCarrierRequest,
	rating *buySideResult,
) {
	if req == nil || rating == nil {
		return
	}

	req.RateMethod = rating.RateMethod
	req.BaseRate = rating.BaseRate
	req.FuelSurcharge = rating.FuelSurcharge
	req.RateQuoteID = rating.RateQuoteID

	if len(rating.Accessorials) > 0 {
		req.Accessorials = rating.Accessorials
	}
}

// enforceMarginFloor stops an assignment the organization said it would not
// accept.
//
// Enforcement is off by default, because a floor that blocks loads on the day
// it is switched on is a floor nobody will keep switched on. When it is on, the
// override is how somebody says "I know, do it anyway" — the same escape the
// insurance warning on this service already has.
func enforceMarginFloor(
	verdict ratetypes.MarginVerdict,
	enforce bool,
	overridden bool,
) error {
	if !verdict.Breached() || !enforce || overridden {
		return nil
	}

	return errortypes.NewBusinessError(
		"This assignment is outside the contract's margin terms: " + verdict.Explanation,
	)
}

// rateBuySide prices this assignment from the carrier's contract.
//
// It runs inside the assignment transaction and before the assignment is built,
// so validation, totals and the rate confirmation all see the contract's
// numbers rather than a second set applied afterwards.
//
// A rating that cannot be produced fails the assignment rather than falling
// back to zero. Somebody asked for the contract's price; a silent zero would
// settle the load for nothing and nobody would notice until the carrier
// invoiced.
func (s *Service) rateBuySide(
	ctx context.Context,
	req *repositories.AssignMoveToCarrierRequest,
	entity *shipment.Shipment,
) error {
	if !shouldAutoRate(req) || s.rateEngine == nil {
		return nil
	}

	sellTotal := sellTotalOf(entity)

	rated, err := s.rateEngine.RateShipment(ctx, &services.RateShipmentRequest{
		Shipment:       entity,
		TenantInfo:     req.TenantInfo,
		PartyType:      rateagreement.PartyTypeCarrier,
		PartyID:        req.CarrierID,
		BillingControl: s.billingControl(ctx, req.TenantInfo),
		SellTotal:      sellTotal,
		Purpose:        ratequote.PurposeRating,
		Persist:        true,
		UserID:         req.TenantInfo.UserID,
	})
	if err != nil {
		return err
	}

	rating, err := buySideRating(rated, s.carrierAgreement(ctx, req.TenantInfo, rated), sellTotal)
	if err != nil {
		return err
	}

	if err = enforceMarginFloor(
		rating.Margin,
		s.marginFloorEnforced(ctx, req.TenantInfo),
		req.OverrideMarginFloor,
	); err != nil {
		return err
	}

	applyBuySideRating(req, rating)

	return nil
}

// sellTotalOf is what the load is being sold for, which is the number margin is
// measured against.
func sellTotalOf(entity *shipment.Shipment) decimal.NullDecimal {
	if entity == nil {
		return decimal.NullDecimal{}
	}

	return entity.TotalChargeAmount
}

// carrierAgreement reads the contract that priced this carrier, for the
// accessorial schedule and the margin terms.
//
// A contract that cannot be read is logged and treated as absent. The pay is
// already priced by this point, and losing an automatic accessorial is a far
// smaller problem than an assignment that cannot be saved.
func (s *Service) carrierAgreement(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rated *services.RatedShipment,
) *rateagreement.RateAgreement {
	if s.agreementRepo == nil || rated == nil || rated.AgreementID == nil {
		return nil
	}

	if rated.Quote != nil && rated.Quote.Agreement != nil &&
		len(rated.Quote.Agreement.Accessorials) > 0 {
		return rated.Quote.Agreement
	}

	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: *rated.AgreementID,
		TenantInfo:      tenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		s.l.Warn("failed to load carrier agreement for assignment pricing",
			zap.String("agreementId", rated.AgreementID.String()),
			zap.Error(err),
		)

		return agreementFrom(rated)
	}

	return agreement
}

// agreementFrom falls back to whatever the quote already carried, so the margin
// terms survive a failed read even when the accessorial schedule does not.
func agreementFrom(rated *services.RatedShipment) *rateagreement.RateAgreement {
	if rated.Quote == nil {
		return nil
	}

	return rated.Quote.Agreement
}

func (s *Service) billingControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) *tenant.BillingControl {
	if s.billingCtrlRepo == nil {
		return nil
	}

	control, err := s.billingCtrlRepo.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		s.l.Warn("failed to read billing control while pricing a carrier", zap.Error(err))

		return nil
	}

	return control
}

func (s *Service) marginFloorEnforced(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) bool {
	control := s.billingControl(ctx, tenantInfo)

	return control != nil && control.EnforceMarginFloor
}

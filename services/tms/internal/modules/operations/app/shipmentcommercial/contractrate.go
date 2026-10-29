package shipmentcommercial

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// RateAgainstContract asks the rate agreements what they would charge for this
// shipment, without changing it.
//
// Nothing here is written to the shipment. Resolving the contract and adopting
// its answer are deliberately two steps, because they happen at different
// moments and sometimes only the first one happens: the billing panel shows a
// rater what the contract says before anyone commits to it, and a save has to
// know what the contract would have charged in order to record the difference
// when somebody charges something else.
func (c *Calculator) RateAgainstContract(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
	persist bool,
) (*ratequoteports.RatedShipment, error) {
	return c.rateEngine.RateShipment(ctx, &ratequoteports.RateShipmentRequest{
		Shipment:  entity,
		Purpose:   ratequote.PurposeRating,
		PartyType: rateagreement.PartyTypeCustomer,
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: userID,
		},
		BillingControl: c.billingControl(ctx, entity),
		Persist:        persist,
		UserID:         userID,
	})
}

func (c *Calculator) AdoptContractRate(
	_ context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
) {
	if rated == nil || !rated.Outcome.Priced() {
		return
	}

	c.applyQuote(entity, rated)
	entity.MarkAutoRated(c.now())
	clearRateDeparture(entity)
}

// RecordRateDeparture records that this shipment is priced at something other
// than what its contract charges.
//
// The difference is the rate leakage report, and it is stored on the shipment
// that caused it rather than recomputed later: contracts get amended, and a
// figure derived from today's terms would quietly restate what last quarter
// gave away. The quote is restated for the same reason — it is the row the
// reports read, and leaving it claiming the contract's number would count a
// discount as revenue.
func (c *Calculator) recordDeparture(
	entity *shipment.Shipment,
	contractAmount decimal.Decimal,
	quote *ratequote.RateQuote,
	userID pulid.ID,
) {
	entity.ClearAutoRating()

	charged := decimal.Zero
	if entity.FreightChargeAmount.Valid {
		charged = entity.FreightChargeAmount.Decimal
	}

	now := c.now()
	entity.RateOverrideAmount = decimal.NewNullDecimal(charged)
	entity.RateOverrideAt = &now
	if !userID.IsNil() {
		entity.RateOverrideByID = &userID
	}

	if quote == nil {
		return
	}

	quote.Outcome = ratequote.OutcomeManualOverride
	quote.OverrideReason = entity.RateOverrideReason
	quote.ForegoneAmount = decimal.NewNullDecimal(contractAmount.Sub(charged))
	quote.LinehaulAmount = charged
	quote.TotalAmount = charged
	quote.BillingAmount = charged
}

// clearRateDeparture wipes a previous hand-priced departure, which a fresh
// application of the contract has just made untrue.
func clearRateDeparture(entity *shipment.Shipment) {
	entity.RateOverrideAmount = decimal.NullDecimal{}
	entity.RateOverrideReason = ""
	entity.RateOverrideByID = nil
	entity.RateOverrideAt = nil
}

func ContractRateMatches(entity *shipment.Shipment, rated *ratequoteports.RatedShipment) bool {
	if entity == nil || rated == nil || !rated.Outcome.Priced() {
		return false
	}

	if rated.FormulaTemplateID == nil || rated.FormulaTemplateID.IsNil() ||
		entity.FormulaTemplateID != *rated.FormulaTemplateID {
		return false
	}

	return entity.TariffInputs.Equal(rated.TariffInputs)
}

// ContractRating is a rating decision that has been made but not yet written
// down, because the shipment it belongs to does not exist yet.
type ContractRating struct {
	// Adopted is true when the shipment is priced exactly the way the contract
	// said to price it, and false when it charges something else.
	Adopted bool
	// ContractAmount is what the contract charged, which is the figure a
	// departure is measured against.
	ContractAmount decimal.Decimal
	Quote          *ratequote.RateQuote
}

// RateAndAdoptContract prices a shipment being created, and settles what the
// contracts had to do with it.
//
// A new shipment is the one moment a contract gets to speak for itself. What
// happens next is decided by comparing what it said against what the payload
// actually holds: identical means the rater took the contract's rate and the
// shipment is marked as carrying it, different means they were shown the
// contract's rate and charged something else. The contract is consulted exactly
// once, and never again unless somebody asks for it.
//
// The decision is returned rather than written, because writing it needs an id
// the shipment does not have until it has been validated and inserted. Commit
// it with CommitContractRating once it does.
func (c *Calculator) RateAndAdoptContract(
	ctx context.Context,
	entity *shipment.Shipment,
	control *shipmentcontrol.ShipmentControl,
	userID pulid.ID,
) (*ContractRating, error) {
	rated, err := c.RateAgainstContract(ctx, entity, userID, false)
	if err != nil {
		return nil, err
	}

	adopted := ContractRateMatches(entity, rated)

	sync := chargeSyncOptions{detention: true, fuel: true}
	switch {
	case adopted:
		c.AdoptContractRate(ctx, entity, rated)
		sync.rated = rated
		sync.contract = rated
	case rated.Outcome.Priced():
		sync.contract = rated
	}

	totals, err := c.recalculate(ctx, entity, control, userID, sync)
	if err != nil {
		return nil, err
	}
	skipped(rated, totals)

	if !adopted && rated.Outcome.Priced() {
		c.recordDeparture(entity, rated.Amount, rated.Quote, userID)
	}

	return &ContractRating{
		Adopted:        adopted,
		ContractAmount: rated.Amount,
		Quote:          rated.Quote,
	}, nil
}

// CommitContractRating writes the quote for a decision RateAndAdoptContract made.
//
// It runs after the shipment has been inserted because the quote points at the
// shipment through a foreign key. Departure fields are prepared during rating
// so they are included in the initial shipment insert.
func (c *Calculator) CommitContractRating(
	ctx context.Context,
	entity *shipment.Shipment,
	rating *ContractRating,
	userID pulid.ID,
) error {
	if rating == nil {
		return nil
	}

	return c.recordQuote(ctx, entity, rating.Quote)
}

// recordQuote writes the rating decision and points the shipment at it.
//
// The quote is written last, once, holding the numbers the shipment ended up
// with rather than the ones it started the save with. It carries no id until
// then, so everything that references it is stamped here rather than where it
// was built.
func (c *Calculator) recordQuote(
	ctx context.Context,
	entity *shipment.Shipment,
	quote *ratequote.RateQuote,
) error {
	if c.quoteRepo == nil || quote == nil {
		return nil
	}

	if quote.ShipmentID == nil || quote.ShipmentID.IsNil() {
		shipmentID := entity.ID
		quote.ShipmentID = &shipmentID
	}

	recorded, err := c.quoteRepo.Record(ctx, quote)
	if err != nil {
		return err
	}

	if recorded == nil || recorded.ID.IsNil() {
		return nil
	}

	quoteID := recorded.ID
	entity.RateQuoteID = &quoteID

	if entity.RatingDetail != nil {
		entity.RatingDetail.RateQuoteID = quoteID.String()
	}

	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.Owner() == shipment.SystemOwnerAgreement {
			charge.RateQuoteID = &quoteID
		}
	}

	return nil
}

// RecordRateDeparture writes down that a shipment has stopped being priced by
// its contract, and by how much.
//
// The transition is what gets a quote of its own: a superseding row saying the
// contract charged one thing and the customer is being billed another. Later
// edits to an already departed shipment only keep the recorded amount honest —
// they are not a second departure, and a quote for each of them would turn the
// rating history into a keystroke log.
func (c *Calculator) RecordRateDeparture(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
	departedNow bool,
) error {
	if entity.AutoRated {
		return nil
	}

	if !departedNow {
		// A shipment that never carried a contract rate has nothing to depart
		// from, and recording an amount against no contract would put it in the
		// leakage report at its full value.
		if entity.HasRateOverride() && entity.FreightChargeAmount.Valid {
			entity.RateOverrideAmount = entity.FreightChargeAmount
		}

		return nil
	}

	applied := c.appliedQuote(ctx, entity)
	contract := decimal.Zero
	if applied != nil {
		contract = applied.LinehaulAmount
	}

	quote := supersedingQuote(entity, applied, contract, c.now())

	c.recordDeparture(entity, contract, quote, userID)

	return c.recordQuote(ctx, entity, quote)
}

// appliedQuote reads the quote currently governing a shipment.
//
// The contract's number is read back rather than recomputed. Re-resolving now
// would answer against today's terms, and the point of the number is what was
// given away against the terms in force when the shipment was rated.
func (c *Calculator) appliedQuote(
	ctx context.Context,
	entity *shipment.Shipment,
) *ratequote.RateQuote {
	if c.quoteRepo == nil || entity.RateQuoteID == nil || entity.RateQuoteID.IsNil() {
		return nil
	}

	quote, err := c.quoteRepo.GetByID(ctx, &ratequoteports.GetRateQuoteByIDRequest{
		RateQuoteID: *entity.RateQuoteID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		// A departure that cannot read its predecessor is still a departure.
		// Recording it with a zero contract amount understates the leakage,
		// which is a far better failure than refusing the save.
		c.logger.Warn(
			"failed to read the applied quote for a rate departure",
			zap.String("shipmentId", entity.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	return quote
}

// supersedingQuote builds the row that records the departure.
//
// It carries the contract that was walked away from, so the leakage report can
// be read by agreement without joining back through a shipment whose stamp the
// departure has since changed, and it bills in the currency the quote it
// replaces was written in.
func supersedingQuote(
	entity *shipment.Shipment,
	applied *ratequote.RateQuote,
	contract decimal.Decimal,
	now int64,
) *ratequote.RateQuote {
	quote := &ratequote.RateQuote{
		OrganizationID:      entity.OrganizationID,
		BusinessUnitID:      entity.BusinessUnitID,
		PartyType:           rateagreement.PartyTypeCustomer,
		PartyID:             entity.CustomerID,
		Purpose:             ratequote.PurposeRating,
		Outcome:             ratequote.OutcomeManualOverride,
		Status:              ratequote.StatusApplied,
		RateAgreementID:     entity.RateAgreementID,
		RateAgreementRuleID: entity.RateAgreementRuleID,
		LinehaulAmount:      contract,
		TotalAmount:         contract,
		BillingAmount:       contract,
		AsOf:                now,
		RatedAt:             now,
	}

	if applied != nil {
		quote.Currency = applied.Currency
		quote.BillingCurrency = applied.BillingCurrency
		quote.FormulaTemplateID = applied.FormulaTemplateID
		quote.SpecificityScore = applied.SpecificityScore
		quote.EngineVersion = applied.EngineVersion
	}

	return quote
}

// AdoptAndRecordContractRate seats a contract's answer on a shipment and writes
// the quote that records it.
//
// It is the deliberate re-rate: the contract has already been resolved by the
// caller, and everything from here is the same work a new shipment does when it
// adopts one. The recalculation runs in between because the accessorials the
// contract just placed change the totals, and the quote is written last so it
// holds what the shipment ended up charging.
func (c *Calculator) AdoptAndRecordContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
	control *shipmentcontrol.ShipmentControl,
	userID pulid.ID,
) error {
	if err := c.AdoptAndRecalculateContractRate(ctx, entity, rated, control, userID); err != nil {
		return err
	}

	return c.RecordContractQuote(ctx, entity, rated)
}

func (c *Calculator) AdoptAndRecalculateContractRate(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
	control *shipmentcontrol.ShipmentControl,
	userID pulid.ID,
) error {
	c.AdoptContractRate(ctx, entity, rated)

	sync := chargeSyncOptions{detention: true, fuel: true}
	if rated != nil && rated.Outcome.Priced() {
		sync.rated = rated
		sync.contract = rated
	}
	totals, err := c.recalculate(ctx, entity, control, userID, sync)
	if err != nil {
		return err
	}
	skipped(rated, totals)
	return nil
}

func skipped(rated *ratequoteports.RatedShipment, totals *commercialTotals) {
	if rated == nil || rated.Quote == nil || rated.Quote.Trace == nil || totals == nil ||
		totals.pass == nil {
		return
	}
	for _, skip := range totals.pass.skipped {
		rated.Quote.Trace.Skipped = append(rated.Quote.Trace.Skipped, ratetypes.Skip{
			Code:   skip.Code,
			Source: skip.Source,
		})
	}
}

func (c *Calculator) RecordContractQuote(
	ctx context.Context,
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
) error {
	if rated == nil {
		return nil
	}

	return c.recordQuote(ctx, entity, rated.Quote)
}

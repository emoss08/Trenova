//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentcommercial

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	skipSourceAgreement = "RateAgreement"
	skipSourceDetention = "Detention"
	skipSourceFuel      = "FuelSurcharge"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	RateEngine         ratequoteports.RateEngine
	AccessorialRepo    accessorialchargeports.AccessorialChargeRepository
	AgreementRepo      rateagreementports.RateAgreementRepository
	QuoteRepo          ratequoteports.RateQuoteRepository
	BillingControlRepo tenantports.BillingControlRepository
	FuelSurcharge      fuelsurchargeports.FuelSurchargeResolver
	DetentionEngine    *detentionservice.Service
	Hydrator           ratequoteports.ShipmentRatingHydrator
}

type Calculator struct {
	logger             *zap.Logger
	rateEngine         ratequoteports.RateEngine
	accessorialRepo    accessorialchargeports.AccessorialChargeRepository
	agreementRepo      rateagreementports.RateAgreementRepository
	quoteRepo          ratequoteports.RateQuoteRepository
	billingControlRepo tenantports.BillingControlRepository
	fuelSurcharge      fuelsurchargeports.FuelSurchargeResolver
	detentionEngine    *detentionservice.Service
	hydrator           ratequoteports.ShipmentRatingHydrator
	now                func() int64
}

func New(p Params) *Calculator {
	return &Calculator{
		logger:             p.Logger.Named("service.shipmentcommercial"),
		rateEngine:         p.RateEngine,
		accessorialRepo:    p.AccessorialRepo,
		agreementRepo:      p.AgreementRepo,
		quoteRepo:          p.QuoteRepo,
		billingControlRepo: p.BillingControlRepo,
		fuelSurcharge:      p.FuelSurcharge,
		detentionEngine:    p.DetentionEngine,
		hydrator:           p.Hydrator,
		now:                timeutils.NowUnix,
	}
}

type chargeSyncOptions struct {
	detention bool
	fuel      bool
	rated     *ratequoteports.RatedShipment
	contract  *ratequoteports.RatedShipment
}

type commercialTotals struct {
	freight decimal.Decimal
	other   decimal.Decimal
	detail  *shipment.RatingDetail
	pass    *ratingPass
}

func (c *Calculator) Recalculate(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) error {
	_, err := c.recalculate(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{detention: true, fuel: true},
	)
	return err
}

func (c *Calculator) recalculate(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
	sync chargeSyncOptions,
) (*commercialTotals, error) {
	totals, err := c.calculateCommercialTotals(ctx, entity, control, userID, sync)
	if err != nil {
		return nil, err
	}

	entity.OtherChargeAmount = decimal.NewNullDecimal(totals.other)
	entity.TotalChargeAmount = decimal.NewNullDecimal(totals.freight.Add(totals.other))
	entity.RatingDetail = totals.detail

	return totals, nil
}

func (c *Calculator) CalculateTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
) (*shipmentports.ShipmentTotalsResponse, error) {
	totals, err := c.calculateCommercialTotals(
		ctx,
		entity,
		control,
		userID,
		chargeSyncOptions{fuel: true},
	)
	if err != nil {
		return nil, err
	}

	return &shipmentports.ShipmentTotalsResponse{
		FreightChargeAmount: totals.freight,
		OtherChargeAmount:   totals.other,
		TotalChargeAmount:   totals.freight.Add(totals.other),
		FuelSurcharge:       findGeneratedFuelSurchargeCharge(entity),
	}, nil
}

func (c *Calculator) hydrate(ctx context.Context, entity *shipment.Shipment) error {
	if c.hydrator == nil || entity == nil {
		return nil
	}

	return c.hydrator.HydrateShipment(ctx, &ratequoteports.HydrateShipmentRequest{
		Shipment: entity,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func findGeneratedFuelSurchargeCharge(entity *shipment.Shipment) *shipment.AdditionalCharge {
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil {
			return charge
		}
	}
	return nil
}

func (c *Calculator) calculateCommercialTotals(
	ctx context.Context,
	entity *shipment.Shipment,
	control *tenant.ShipmentControl,
	userID pulid.ID,
	sync chargeSyncOptions,
) (*commercialTotals, error) {
	ctx = ratetablecache.With(ctx)

	if err := c.hydrate(ctx, entity); err != nil {
		return nil, err
	}

	own, err := c.rateOwn(ctx, entity, userID, sync.rated)
	if err != nil {
		return nil, err
	}

	pass, err := c.newRatingPass(ctx, entity, own)
	if err != nil {
		return nil, err
	}

	if !entity.RateLocked {
		if err = c.applyBill(ctx, entity, own, pass); err != nil {
			return nil, err
		}
	} else if len(entity.FreightLines) == 0 {
		entity.SetFreightLines(entity.ManualFreightLines(i18n.T(ctx, "Freight charge")))
	}

	contract := sync.contract
	switch {
	case entity.RateLocked:
	case contract != nil:
		if err = c.syncAgreementAccessorials(ctx, entity, contract, pass); err != nil {
			return nil, err
		}
	default:
		pass.dropBilledAgreementCharges(entity)
	}

	if sync.detention {
		if err = c.syncDetentionCharge(ctx, entity, control, pass); err != nil {
			return nil, err
		}
	}

	freight := freightOf(entity)

	if sync.fuel {
		agreement := c.fuelAgreement(ctx, entity, contract)
		if err = c.syncFuelSurcharge(ctx, entity, freight, agreement, pass); err != nil {
			return nil, err
		}
	}

	other, err := shipment.AdditionalChargeTotals(ctx, entity.AdditionalCharges, freight)
	if err != nil {
		return nil, err
	}

	return &commercialTotals{
		freight: freight,
		other:   other.Sum,
		detail:  c.ratingDetail(entity, own, pass),
		pass:    pass,
	}, nil
}

func freightOf(entity *shipment.Shipment) decimal.Decimal {
	if entity.FreightChargeAmount.Valid {
		return entity.FreightChargeAmount.Decimal
	}
	return decimal.Zero
}

func (c *Calculator) fuelAgreement(
	ctx context.Context,
	entity *shipment.Shipment,
	contract *ratequoteports.RatedShipment,
) *rateagreement.RateAgreement {
	if contract != nil {
		return contract.Agreement
	}
	return c.loadAgreementByID(ctx, entity, entity.RateAgreementID)
}

func (c *Calculator) rateOwn(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
	rated *ratequoteports.RatedShipment,
) (*ratequoteports.RatedShipment, error) {
	if entity.RatedByHand() {
		return nil, nil //nolint:nilnil // a shipment rated by hand has no tariff rating
	}
	if rated != nil {
		return rated, nil
	}

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
		FormulaOnly:    true,
		Persist:        false,
		UserID:         userID,
	})
}

func (c *Calculator) ratingDetail(
	entity *shipment.Shipment,
	own *ratequoteports.RatedShipment,
	pass *ratingPass,
) *shipment.RatingDetail {
	if entity.RateLocked {
		return entity.RatingDetail
	}
	if own == nil {
		return &shipment.RatingDetail{
			Source:  shipment.RatingSourceManual,
			RatedAt: c.now(),
		}
	}

	detail := ratingDetailFromQuote(own)
	detail.Skipped = pass.skipped
	carryContractProvenance(entity, detail)
	detail.CarryReviewApproval(entity.RatingDetail)
	return detail
}

func carryContractProvenance(entity *shipment.Shipment, detail *shipment.RatingDetail) {
	previous := entity.RatingDetail
	if detail == nil || previous == nil || !entity.AutoRated {
		return
	}

	if detail.RateQuoteID == "" {
		detail.RateQuoteID = previous.RateQuoteID
	}
	if detail.AgreementName == "" {
		detail.AgreementName = previous.AgreementName
	}
	if detail.RuleLabel == "" {
		detail.RuleLabel = previous.RuleLabel
	}
	if detail.Explanation == "" {
		detail.Explanation = previous.Explanation
	}
}

func (c *Calculator) applyQuote(
	entity *shipment.Shipment,
	rated *ratequoteports.RatedShipment,
) {
	seedBillToFromAgreement(entity, rated)
	entity.RateAgreementID = rated.AgreementID
	entity.RateAgreementRuleID = rated.RuleID

	if rated.Quote != nil && !rated.Quote.ID.IsNil() {
		quoteID := rated.Quote.ID
		entity.RateQuoteID = &quoteID
	}

	if rated.FormulaTemplateID != nil && !rated.FormulaTemplateID.IsNil() {
		entity.FormulaTemplateID = *rated.FormulaTemplateID
	}
	entity.TariffInputs = rated.TariffInputs.Clone()

	detail := ratingDetailFromQuote(rated)
	detail.CarryReviewApproval(entity.RatingDetail)
	entity.RatingDetail = detail
}

func seedBillToFromAgreement(entity *shipment.Shipment, rated *ratequoteports.RatedShipment) {
	if entity == nil || rated == nil || rated.BillToCustomerID == nil ||
		rated.BillToCustomerID.IsNil() || rated.AgreementID == nil {
		return
	}
	if entity.HasExplicitBillTo() {
		return
	}
	if entity.RateAgreementID != nil && *entity.RateAgreementID == *rated.AgreementID {
		return
	}
	if *rated.BillToCustomerID == entity.CustomerID {
		return
	}
	billTo := *rated.BillToCustomerID
	entity.BillToCustomerID = &billTo
}

func ratingDetailFromQuote(rated *ratequoteports.RatedShipment) *shipment.RatingDetail {
	detail := &shipment.RatingDetail{
		Bill:                rated.Bill,
		Refusal:             rated.Refusal,
		Choices:             rated.Choices,
		Source:              string(rated.Outcome),
		FormulaTemplateName: rated.FormulaTemplateName,
		VersionNumber:       rated.TemplateVersion,
		Inputs:              rated.TariffInputs,
	}
	if rated.FormulaTemplateID != nil {
		detail.FormulaTemplateID = rated.FormulaTemplateID.String()
	}
	if rated.AgreementID != nil {
		detail.AgreementID = rated.AgreementID.String()
	}
	if rated.RuleID != nil {
		detail.RuleID = rated.RuleID.String()
	}
	if rated.Agreement != nil {
		detail.AgreementName = rated.Agreement.Name
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

	if trace := quote.Trace; trace != nil {
		detail.Trace = ratingTrace(trace)
		if winner := trace.Winner(); winner != nil {
			detail.AgreementName = winner.AgreementName
			detail.RuleLabel = winner.RuleLabel
		}
	}

	return detail
}

func ratingTrace(trace *ratetypes.Trace) *shipment.RatingTrace {
	if trace.Statements.IsNone() {
		return nil
	}
	return &shipment.RatingTrace{Statements: trace.Statements, Lookups: trace.Lookups}
}

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
	pass *ratingPass,
) error {
	if entity == nil || control == nil {
		return nil
	}

	if control.UseDetentionPolicyEngine {
		return c.syncDetentionFromPolicyEngine(ctx, entity, pass)
	}

	if !control.TrackDetentionTime ||
		!control.AutoGenerateDetentionCharges ||
		control.DetentionChargeID == nil ||
		control.DetentionChargeID.IsNil() {
		return nil
	}

	accessorial, err := c.accessorialRepo.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID: *control.DetentionChargeID,
			TenantInfo: &pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
	if err != nil {
		return err
	}

	if pass.skips(accessorial.ID, skipSourceDetention) {
		removeGeneratedDetentionCharges(entity, accessorial.ID)
		return nil
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
	pass *ratingPass,
) error {
	if entity == nil || c.fuelSurcharge == nil {
		return nil
	}

	if entity.FuelSurchargeLocked {
		return nil
	}

	accessorialTotal, err := nonFuelSurchargeChargeTotal(ctx, entity, baseCharge)
	if err != nil {
		return err
	}

	resolved, err := c.fuelSurcharge.ResolveShipmentCharge(
		ctx,
		&fuelsurchargeports.ResolveShipmentChargeRequest{
			Shipment:         entity,
			Linehaul:         baseCharge,
			AccessorialTotal: accessorialTotal,
			Override:         fuelOverride(agreement),
		},
	)
	if err != nil {
		return err
	}

	if resolved == nil || resolved.Amount.LessThanOrEqual(decimal.Zero) ||
		pass.skips(resolved.AccessorialChargeID, skipSourceFuel) {
		removeGeneratedFuelSurchargeCharges(entity)
		return nil
	}

	ensureGeneratedFuelSurchargeCharge(entity, resolved)
	return nil
}

func nonFuelSurchargeChargeTotal(
	ctx context.Context,
	entity *shipment.Shipment,
	baseCharge decimal.Decimal,
) (decimal.Decimal, error) {
	charges := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))
	for _, charge := range entity.AdditionalCharges {
		if charge == nil || (charge.IsSystemGenerated && charge.FuelSurchargeProgramID != nil) {
			continue
		}
		charges = append(charges, charge)
	}

	totals, err := shipment.AdditionalChargeTotals(ctx, charges, baseCharge)
	if err != nil {
		return decimal.Zero, err
	}

	return totals.Sum, nil
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
	resolved *fuelsurchargeports.ResolvedFuelSurcharge,
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

func isDetentionChargeFor(charge *shipment.AdditionalCharge, accessorialID pulid.ID) bool {
	if charge == nil || !charge.IsSystemGenerated || charge.AccessorialChargeID != accessorialID {
		return false
	}

	owner := charge.Owner()

	return owner == shipment.SystemOwnerDetention || owner == shipment.SystemOwnerNone
}

func preferDetentionCharge(candidate, current *shipment.AdditionalCharge) bool {
	switch {
	case candidate.ID.IsNil() != current.ID.IsNil():
		return current.ID.IsNil()
	case candidate.CreatedAt != current.CreatedAt:
		return candidate.CreatedAt < current.CreatedAt
	default:
		return candidate.ID.String() < current.ID.String()
	}
}

func takeDetentionCharge(
	charges []*shipment.AdditionalCharge,
	accessorialID pulid.ID,
) (*shipment.AdditionalCharge, []*shipment.AdditionalCharge) {
	var found *shipment.AdditionalCharge
	rest := make([]*shipment.AdditionalCharge, 0, len(charges))

	for _, charge := range charges {
		if charge == nil {
			continue
		}

		if !isDetentionChargeFor(charge, accessorialID) {
			rest = append(rest, charge)
			continue
		}

		if found == nil || preferDetentionCharge(charge, found) {
			found = charge
		}
	}

	return found, rest
}

func removeGeneratedDetentionCharges(entity *shipment.Shipment, detentionChargeID pulid.ID) {
	_, rest := takeDetentionCharge(entity.AdditionalCharges, detentionChargeID)
	entity.AdditionalCharges = rest
}

func ensureGeneratedDetentionCharge(
	entity *shipment.Shipment,
	accessorial *accessorialcharge.AccessorialCharge,
	unit int16,
) {
	generated, rest := takeDetentionCharge(entity.AdditionalCharges, accessorial.ID)
	if generated == nil {
		generated = &shipment.AdditionalCharge{}
	}

	generated.OrganizationID = entity.OrganizationID
	generated.BusinessUnitID = entity.BusinessUnitID
	generated.ShipmentID = entity.ID
	generated.AccessorialChargeID = accessorial.ID
	generated.IsSystemGenerated = true
	generated.IsDetention = true
	generated.Method = accessorial.Method
	generated.Amount = accessorial.Amount
	generated.Unit = unit
	generated.DetentionOccurrenceIDs = nil

	entity.AdditionalCharges = append(rest, generated)
}

// syncDetentionFromPolicyEngine rebuilds detention charges from resolved policy
// occurrences. The billable occurrences behind one accessorial fold into one
// system charge carrying the amounts the engine already computed, so the
// invoice line and the stored calculation traces can never disagree.
func (c *Calculator) syncDetentionFromPolicyEngine(
	ctx context.Context,
	entity *shipment.Shipment,
	pass *ratingPass,
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

	reconcileDetentionOccurrenceCharges(entity, pass.billableOccurrences(result.Occurrences))

	return nil
}

// reconcileDetentionOccurrenceCharges rewrites the shipment's detention charges
// so exactly one charge survives per accessorial the engine bills, carrying the
// sum of every billable occurrence behind it. The existing row is reused rather
// than replaced: a charge that keeps its identity keeps its audit trail, and
// the occurrences it bills are pointed at it when the shipment is saved.
func reconcileDetentionOccurrenceCharges(
	entity *shipment.Shipment,
	occurrences []*detention.DetentionOccurrence,
) {
	groups := groupChargeableOccurrences(occurrences)

	rest := entity.AdditionalCharges
	generated := make([]*shipment.AdditionalCharge, 0, len(groups))

	for i := range groups {
		var existing *shipment.AdditionalCharge
		existing, rest = takeDetentionCharge(rest, groups[i].accessorialID)
		generated = append(generated, detentionGroupCharge(entity, &groups[i], existing))
	}

	kept := make([]*shipment.AdditionalCharge, 0, len(rest)+len(generated))
	for _, charge := range rest {
		if charge == nil || charge.Owner() == shipment.SystemOwnerDetention {
			continue
		}
		kept = append(kept, charge)
	}

	entity.AdditionalCharges = append(kept, generated...)
}

type detentionChargeGroup struct {
	accessorialID pulid.ID
	amount        decimal.Decimal
	occurrenceIDs []pulid.ID
}

func groupChargeableOccurrences(
	occurrences []*detention.DetentionOccurrence,
) []detentionChargeGroup {
	groups := make([]detentionChargeGroup, 0, 1)
	index := make(map[pulid.ID]int, 1)

	for _, occurrence := range occurrences {
		if !occurrence.Chargeable() {
			continue
		}

		accessorialID := occurrence.PolicySnapshot.AccessorialChargeID

		i, ok := index[accessorialID]
		if !ok {
			i = len(groups)
			index[accessorialID] = i
			groups = append(groups, detentionChargeGroup{
				accessorialID: accessorialID,
				amount:        decimal.Zero,
				occurrenceIDs: make([]pulid.ID, 0, 2),
			})
		}

		groups[i].amount = groups[i].amount.Add(occurrence.BillableAmount)
		groups[i].occurrenceIDs = append(groups[i].occurrenceIDs, occurrence.ID)
	}

	return groups
}

func detentionGroupCharge(
	entity *shipment.Shipment,
	group *detentionChargeGroup,
	existing *shipment.AdditionalCharge,
) *shipment.AdditionalCharge {
	charge := existing
	if charge == nil {
		charge = &shipment.AdditionalCharge{}
	}

	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.IsSystemGenerated = true
	charge.IsDetention = true
	charge.AccessorialChargeID = group.accessorialID
	charge.Method = accessorialcharge.MethodFlat
	charge.Amount = group.amount
	charge.Unit = 1
	charge.DetentionOccurrenceIDs = group.occurrenceIDs

	return charge
}

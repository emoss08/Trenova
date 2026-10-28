package carrierassignmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// buySideResult is what a carrier's contract said this move is worth, in the
// shape the assignment request already takes.
type buySideResult struct {
	RateMethod    shipment.CarrierRateMethod
	BaseRate      decimal.Decimal
	FuelSurcharge decimal.Decimal
	Accessorials  []carrierassignmentports.CarrierAccessorialInput
	RateQuoteID   *pulid.ID
	Margin        ratetypes.MarginVerdict
}

// shouldAutoRate reports whether the contract should price this assignment.
//
// A rate somebody typed is a decision, and auto-rating over the top of it would
// be the system overruling a negotiated number. That is precisely what makes
// people switch auto-rating off, so a typed rate always wins.
func shouldAutoRate(req *carrierassignmentports.AssignMoveToCarrierRequest) bool {
	return req != nil && req.AutoRate && req.BaseRate.IsZero()
}

type buySideParts struct {
	rated     *ratequoteports.RatedShipment
	agreement *rateagreement.RateAgreement
	entity    *shipment.Shipment
	sellTotal decimal.NullDecimal
	at        int64
}

func (s *Service) buySideRating(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	parts *buySideParts,
) (*buySideResult, error) {
	rated := parts.rated
	if rated == nil || !rated.Outcome.Priced() {
		return nil, errortypes.NewBusinessError(
			"No carrier agreement covers this lane, so the carrier's pay could not be priced",
		)
	}

	catalog, err := s.buySideCatalog(ctx, tenantInfo, parts)
	if err != nil {
		return nil, err
	}

	result := &buySideResult{
		RateMethod:   shipment.CarrierRateMethodFlat,
		BaseRate:     rated.Amount,
		RateQuoteID:  quoteIDOf(rated),
		Accessorials: []carrierassignmentports.CarrierAccessorialInput{},
	}
	billed, err := s.applyTariffCharges(ctx, tenantInfo, result, rated, catalog)
	if err != nil {
		return nil, err
	}

	contract, err := contractAccessorials(ctx, parts, catalog, billed)
	if err != nil {
		return nil, err
	}
	result.Accessorials = append(result.Accessorials, contract...)

	if err = s.applyFuelProgram(ctx, parts, result, catalog, billed); err != nil {
		return nil, err
	}

	result.Margin = ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      valueOf(parts.sellTotal),
		Buy:       result.total(),
		Floor:     floorOf(parts.agreement),
		MaxPayPct: ceilingOf(parts.agreement),
	})

	return result, nil
}

func (r *buySideResult) total() decimal.Decimal {
	total := r.BaseRate.Add(r.FuelSurcharge)
	for _, accessorial := range r.Accessorials {
		total = total.Add(accessorial.Amount)
	}
	return total
}

type buySideCatalog struct {
	byCode map[string]*accessorialcharge.AccessorialCharge
	codeOf map[pulid.ID]string
}

func (c *buySideCatalog) code(id pulid.ID) string {
	if c == nil {
		return ""
	}
	return c.codeOf[id]
}

func (s *Service) buySideCatalog(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	parts *buySideParts,
) (*buySideCatalog, error) {
	coded := parts.rated.Bill != nil && len(parts.rated.Bill.CodedLines()) > 0
	scheduled := parts.agreement != nil &&
		(len(parts.agreement.Accessorials) > 0 || parts.agreement.FuelBinding != nil)
	if !coded && !scheduled {
		return &buySideCatalog{}, nil
	}
	if s.accessorialRepo == nil {
		return nil, errortypes.NewBusinessError(
			"The accessorial catalog is not available, so the carrier's pay could not be priced",
		)
	}
	charges, err := s.accessorialRepo.ListCatalog(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	out := &buySideCatalog{
		byCode: make(map[string]*accessorialcharge.AccessorialCharge, len(charges)),
		codeOf: make(map[pulid.ID]string, len(charges)),
	}
	for _, charge := range charges {
		code := accessorialcharge.NormalizeCode(charge.Code)
		out.byCode[code] = charge
		out.codeOf[charge.ID] = code
	}
	return out, nil
}

func (s *Service) applyTariffCharges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	result *buySideResult,
	rated *ratequoteports.RatedShipment,
	catalog *buySideCatalog,
) (map[string]struct{}, error) {
	billed := map[string]struct{}{}
	if rated.Bill == nil {
		return billed, nil
	}
	coded := rated.Bill.CodedLines()
	if len(coded) == 0 {
		return billed, nil
	}
	fuelCodes, err := s.fuelCodes(ctx, tenantInfo, catalog)
	if err != nil {
		return nil, err
	}
	for i := range coded {
		line := &coded[i]
		code := accessorialcharge.NormalizeCode(line.Code)
		billed[code] = struct{}{}
		if !line.Amount.IsPositive() {
			continue
		}
		if _, fuel := fuelCodes[code]; fuel {
			result.FuelSurcharge = result.FuelSurcharge.Add(line.Amount)
			continue
		}
		definition, ok := catalog.byCode[code]
		if !ok {
			return nil, errortypes.NewBusinessError(
				"The tariff billed the accessorial code {0}, which is not in the accessorial charge catalog",
				line.Code,
			)
		}
		chargeID := definition.ID
		description := line.Label
		if description == "" {
			description = accessorialDescription(definition)
		}
		result.Accessorials = append(
			result.Accessorials,
			carrierassignmentports.CarrierAccessorialInput{
				AccessorialChargeID: &chargeID,
				Description:         description,
				Amount:              line.Amount,
			},
		)
	}
	return billed, nil
}

func (s *Service) fuelCodes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	catalog *buySideCatalog,
) (map[string]struct{}, error) {
	codes := map[string]struct{}{}
	if s.fuelResolver == nil {
		return codes, nil
	}
	ids, err := s.fuelResolver.FuelChargeIDs(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if code := catalog.code(id); code != "" {
			codes[code] = struct{}{}
		}
	}
	return codes, nil
}

func contractAccessorials(
	ctx context.Context,
	parts *buySideParts,
	catalog *buySideCatalog,
	billed map[string]struct{},
) ([]carrierassignmentports.CarrierAccessorialInput, error) {
	if parts.agreement == nil {
		return nil, nil
	}
	scoped := parts.agreement.AutoApplyAccessorials(rateagreement.AccessorialFacts{
		At:             parts.at,
		ServiceTypeID:  parts.entity.ServiceTypeID,
		ShipmentTypeID: parts.entity.ShipmentTypeID,
	})
	applicable := make([]*rateagreement.RateAgreementAccessorial, 0, len(scoped))
	for _, row := range scoped {
		if strings.TrimSpace(row.Condition) != "" && !parts.rated.Conditions[row.ID] {
			continue
		}
		if _, tariffBilled := billed[catalog.code(row.AccessorialChargeID)]; tariffBilled {
			continue
		}
		applicable = append(applicable, row)
	}

	priced, err := shipment.PriceAgreementAccessorials(
		ctx,
		parts.entity.AccessorialMeasures(),
		applicable,
		parts.rated.Amount,
	)
	if err != nil {
		return nil, err
	}
	out := make([]carrierassignmentports.CarrierAccessorialInput, 0, len(priced))
	for _, entry := range priced {
		chargeID := entry.Accessorial.AccessorialChargeID
		definition := entry.Accessorial.AccessorialCharge
		if definition == nil && catalog != nil {
			definition = catalog.byCode[catalog.code(chargeID)]
		}
		out = append(out, carrierassignmentports.CarrierAccessorialInput{
			AccessorialChargeID: &chargeID,
			Description:         accessorialDescription(definition),
			Amount:              entry.Charge.Amount,
		})
	}
	return out, nil
}

func (s *Service) applyFuelProgram(
	ctx context.Context,
	parts *buySideParts,
	result *buySideResult,
	catalog *buySideCatalog,
	billed map[string]struct{},
) error {
	override := carrierFuelOverride(parts.agreement)
	if override == nil || override.Waived || s.fuelResolver == nil {
		return nil
	}
	accessorialTotal := decimal.Zero
	for _, accessorial := range result.Accessorials {
		accessorialTotal = accessorialTotal.Add(accessorial.Amount)
	}
	resolved, err := s.fuelResolver.ResolveShipmentCharge(
		ctx,
		&fuelsurchargeports.ResolveShipmentChargeRequest{
			Shipment:         parts.entity,
			Linehaul:         parts.rated.Amount,
			AccessorialTotal: accessorialTotal,
			Override:         override,
		},
	)
	if err != nil {
		return err
	}
	if resolved == nil || !resolved.Amount.IsPositive() {
		return nil
	}
	if _, tariffBilled := billed[catalog.code(resolved.AccessorialChargeID)]; tariffBilled {
		return nil
	}
	result.FuelSurcharge = result.FuelSurcharge.Add(resolved.Amount)
	return nil
}

func carrierFuelOverride(
	agreement *rateagreement.RateAgreement,
) *fuelsurchargeports.FuelProgramOverride {
	if agreement == nil || agreement.FuelBinding == nil {
		return nil
	}
	binding := agreement.FuelBinding
	return &fuelsurchargeports.FuelProgramOverride{
		AgreementID:   agreement.ID,
		ProgramID:     binding.FuelSurchargeProgramID,
		Waived:        binding.Waived,
		PegPrice:      binding.PegPriceOverride,
		IncrementRate: binding.IncrementRateOverride,
		CapAmount:     binding.CapAmount,
	}
}

func quoteIDOf(rated *ratequoteports.RatedShipment) *pulid.ID {
	if rated.Quote == nil || rated.Quote.ID.IsNil() {
		return nil
	}

	id := rated.Quote.ID

	return &id
}

func accessorialDescription(definition *accessorialcharge.AccessorialCharge) string {
	if definition == nil {
		return ""
	}

	if definition.Description != "" {
		return definition.Description
	}

	return definition.Code
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
	req *carrierassignmentports.AssignMoveToCarrierRequest,
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
// accept. The refusal names no figure: whoever assigns the carrier may not
// hold the margins field group.
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
		"This assignment is outside the contract's margin terms; override the margin floor to assign it anyway",
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
	req *carrierassignmentports.AssignMoveToCarrierRequest,
	entity *shipment.Shipment,
) error {
	if !shouldAutoRate(req) || s.rateEngine == nil {
		return nil
	}

	sellTotal := sellTotalOf(entity)

	rated, err := s.rateEngine.RateShipment(ctx, &ratequoteports.RateShipmentRequest{
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

	rating, err := s.buySideRating(ctx, req.TenantInfo, &buySideParts{
		rated:     rated,
		agreement: s.carrierAgreement(ctx, req.TenantInfo, rated),
		entity:    entity,
		sellTotal: sellTotal,
		at:        buySideDate(entity),
	})
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

func buySideDate(entity *shipment.Shipment) int64 {
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		return *entity.ActualShipDate
	}
	if entity.CreatedAt > 0 {
		return entity.CreatedAt
	}
	return timeutils.NowUnix()
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
	rated *ratequoteports.RatedShipment,
) *rateagreement.RateAgreement {
	if s.agreementRepo == nil || rated == nil || rated.AgreementID == nil {
		return nil
	}

	if rated.Agreement != nil {
		return rated.Agreement
	}

	agreement, err := s.agreementRepo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
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
func agreementFrom(rated *ratequoteports.RatedShipment) *rateagreement.RateAgreement {
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

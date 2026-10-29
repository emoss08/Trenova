package shipmentcommercial

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ratingPass struct {
	billed  map[string]struct{}
	codeOf  map[pulid.ID]string
	byCode  map[string]*accessorialcharge.AccessorialCharge
	skipped []shipment.RatingSkip
}

func (p *ratingPass) skips(accessorialID pulid.ID, source string) bool {
	if p == nil || len(p.billed) == 0 {
		return false
	}
	code, ok := p.codeOf[accessorialID]
	if !ok {
		return false
	}
	if _, billed := p.billed[code]; !billed {
		return false
	}
	for _, skip := range p.skipped {
		if skip.Code == code && skip.Source == source {
			return true
		}
	}
	p.skipped = append(p.skipped, shipment.RatingSkip{Code: code, Source: source})
	return true
}

func (p *ratingPass) billableOccurrences(
	occurrences []*detention.DetentionOccurrence,
) []*detention.DetentionOccurrence {
	if p == nil || len(p.billed) == 0 {
		return occurrences
	}
	kept := make([]*detention.DetentionOccurrence, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.Chargeable() &&
			p.skips(occurrence.PolicySnapshot.AccessorialChargeID, skipSourceDetention) {
			continue
		}
		kept = append(kept, occurrence)
	}
	return kept
}

func (c *Calculator) newRatingPass(
	ctx context.Context,
	entity *shipment.Shipment,
	own *ratequoteports.RatedShipment,
) (*ratingPass, error) {
	pass := &ratingPass{billed: map[string]struct{}{}}
	if own == nil || own.Bill == nil || len(own.Bill.CodedLines()) == 0 {
		return pass, nil
	}
	if err := c.loadCatalog(ctx, entity, pass); err != nil {
		return nil, err
	}
	for _, line := range own.Bill.CodedLines() {
		pass.billed[accessorialcharge.NormalizeCode(line.Code)] = struct{}{}
	}
	return pass, nil
}

func (c *Calculator) loadCatalog(
	ctx context.Context,
	entity *shipment.Shipment,
	pass *ratingPass,
) error {
	if pass.byCode != nil {
		return nil
	}
	catalog, err := c.accessorialRepo.ListCatalog(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return err
	}
	pass.codeOf = make(map[pulid.ID]string, len(catalog))
	pass.byCode = make(map[string]*accessorialcharge.AccessorialCharge, len(catalog))
	for _, charge := range catalog {
		code := accessorialcharge.NormalizeCode(charge.Code)
		pass.codeOf[charge.ID] = code
		pass.byCode[code] = charge
	}
	return nil
}

func (p *ratingPass) definition(accessorialID pulid.ID) *accessorialcharge.AccessorialCharge {
	code, ok := p.codeOf[accessorialID]
	if !ok {
		return nil
	}
	return p.byCode[code]
}

func (c *Calculator) applyBill(
	ctx context.Context,
	entity *shipment.Shipment,
	own *ratequoteports.RatedShipment,
	pass *ratingPass,
) error {
	if own == nil {
		entity.SetFreightLines(entity.ManualFreightLines(i18n.T(ctx, "Freight charge")))
		removeTariffCharges(entity)
		return nil
	}
	if own.Bill == nil {
		entity.SetFreightLines([]shipment.FreightLine{})
		removeTariffCharges(entity)
		return nil
	}

	entity.SetFreightLines(billFreightLines(ctx, own.Bill))
	return reconcileTariffCharges(entity, own.Bill.CodedLines(), pass)
}

func billFreightLines(ctx context.Context, bill *tarifftypes.Bill) []shipment.FreightLine {
	return shipmentrating.FreightLinesFromBill(bill, func(kind tarifftypes.LineKind) string {
		return adjustmentLabel(ctx, kind)
	})
}

func adjustmentLabel(ctx context.Context, kind tarifftypes.LineKind) string {
	switch kind {
	case tarifftypes.LineMinimum:
		return i18n.T(ctx, "Minimum charge adjustment")
	case tarifftypes.LineMaximum:
		return i18n.T(ctx, "Maximum charge adjustment")
	case tarifftypes.LineRounding:
		return i18n.T(ctx, "Rounding")
	case tarifftypes.LineCharge:
		return i18n.T(ctx, "Freight charge")
	default:
		return i18n.T(ctx, "Freight charge")
	}
}

func reconcileTariffCharges(
	entity *shipment.Shipment,
	lines []tarifftypes.Line,
	pass *ratingPass,
) error {
	existing := make([]*shipment.AdditionalCharge, 0, len(lines))
	kept := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges)+len(lines))
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if charge.Owner() == shipmentvocab.SystemOwnerTariff {
			existing = append(existing, charge)
			continue
		}
		kept = append(kept, charge)
	}

	for index := range lines {
		line := &lines[index]
		definition, ok := pass.byCode[accessorialcharge.NormalizeCode(line.Code)]
		if !ok {
			return unknownCodeError(line.Code)
		}
		if !line.Amount.IsPositive() {
			continue
		}
		var charge *shipment.AdditionalCharge
		if index < len(existing) {
			charge = existing[index]
		} else {
			charge = &shipment.AdditionalCharge{}
		}
		kept = append(kept, tariffCharge(entity, charge, definition, line))
	}

	entity.AdditionalCharges = kept
	return nil
}

func tariffCharge(
	entity *shipment.Shipment,
	charge *shipment.AdditionalCharge,
	definition *accessorialcharge.AccessorialCharge,
	line *tarifftypes.Line,
) *shipment.AdditionalCharge {
	charge.OrganizationID = entity.OrganizationID
	charge.BusinessUnitID = entity.BusinessUnitID
	charge.ShipmentID = entity.ID
	charge.AccessorialChargeID = definition.ID
	charge.AccessorialCharge = definition
	charge.IsSystemGenerated = true
	charge.IsTariff = true
	charge.IsDetention = false
	charge.FuelSurchargeProgramID = nil
	charge.RateAgreementAccessorialID = nil
	charge.FuelSurchargeDetail = nil
	charge.DetentionOccurrenceIDs = nil
	charge.Method = accessorialcharge.MethodFlat
	charge.Amount = line.Amount
	charge.Unit = 1
	charge.Label = line.Label
	charge.Quantity = nullDecimal(line.Quantity)
	charge.Rate = nullDecimal(line.Rate)
	charge.RateUnit = line.RateUnit
	return charge
}

func nullDecimal(value *decimal.Decimal) decimal.NullDecimal {
	if value == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal(*value)
}

func removeTariffCharges(entity *shipment.Shipment) {
	kept := entity.AdditionalCharges[:0]
	for _, charge := range entity.AdditionalCharges {
		if charge != nil && charge.Owner() != shipmentvocab.SystemOwnerTariff {
			kept = append(kept, charge)
		}
	}
	entity.AdditionalCharges = kept
}

package invoicelines

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentrating"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	FreightDescription     = "Freight charge"
	AccessorialDescription = "Accessorial charge"
)

func SignedAmount(billType billingqueue.BillType, amount decimal.Decimal) decimal.Decimal {
	if billType == billingqueue.BillTypeCreditMemo {
		return amount.Neg()
	}

	return amount
}

// ForShipment emits the lines that bill a shipment in full to one payer.
func ForShipment(
	ctx context.Context,
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	startLineNumber int,
) ([]*invoice.InvoiceLine, error) {
	if shp == nil {
		return nil, nil
	}

	resolution, err := chargeallocation.ResolveShares(ctx, shp, nil)
	if err != nil {
		return nil, err
	}

	return ForShipmentShare(billType, shp, resolution.Primary(), startLineNumber), nil
}

// ForShipmentShare emits only the lines one payer is billed for. A
// partial share carries the percent it bills on the line and in its description,
// so the invoice reads correctly on its own.
func ForShipmentShare(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	share *chargeallocation.PayerShare,
	startLineNumber int,
) []*invoice.InvoiceLine {
	if shp == nil || share == nil {
		return nil
	}

	lines := make([]*invoice.InvoiceLine, 0, len(share.Charges)+len(shp.FreightLines))
	freight := shp.FreightChargeAmount.Decimal
	lineNumber := startLineNumber

	for _, charge := range share.Charges {
		switch charge.Kind {
		case shipment.ChargeAllocationKindFreight:
			freightLines := freightLines(billType, shp, charge, lineNumber)
			lines = append(lines, freightLines...)
			lineNumber += len(freightLines)
			continue
		case shipment.ChargeAllocationKindAccessorial:
			if charge.Charge == nil {
				continue
			}
			lines = append(lines, accessorialLine(billType, shp, charge, freight, lineNumber))
		case shipment.ChargeAllocationKindOrderCharge:
			continue
		}
		lineNumber++
	}

	return lines
}

// ForOrderChargeShare emits the line for one payer's share of an order-level
// charge. Order charges carry no leg attribution, which is what puts them in the
// trailing section when the invoice is rendered.
func ForOrderChargeShare(
	billType billingqueue.BillType,
	charge chargeallocation.AllocatedCharge,
	lineNumber int,
) *invoice.InvoiceLine {
	amount := SignedAmount(billType, charge.Amount)
	line := &invoice.InvoiceLine{
		LineNumber:  lineNumber,
		Type:        invoice.InvoiceLineTypeAccessorial,
		Description: ShareDescription(charge.Description, charge),
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   amount,
		Amount:      amount,
	}
	stampShare(line, charge)

	return line
}

// ShareDescription appends the billed share to a description when the line
// bills less than the whole charge.
func ShareDescription(base string, charge chargeallocation.AllocatedCharge) string {
	if !charge.Partial {
		return base
	}
	if charge.Percent.Valid {
		percent := strings.TrimRight(
			strings.TrimRight(charge.Percent.Decimal.StringFixed(2), "0"),
			".",
		)
		return base + " (" + percent + "% share)"
	}

	return base + " (partial share)"
}

func stampShare(line *invoice.InvoiceLine, charge chargeallocation.AllocatedCharge) {
	if !charge.Partial {
		return
	}
	line.AllocationPercent = charge.Percent
	line.ChargeAllocationID = charge.AllocationID
}

func freightLines(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	charge chargeallocation.AllocatedCharge,
	startLineNumber int,
) []*invoice.InvoiceLine {
	billed := shp.BilledFreightLines(FreightDescription)
	amounts := shipmentrating.SplitFreightShare(billed, charge.Amount)
	templateID, templateName := ratedWith(shp)

	out := make([]*invoice.InvoiceLine, 0, len(billed))
	for i := range billed {
		freight := &billed[i]
		amount := SignedAmount(billType, amounts[i])
		label := strings.TrimSpace(freight.Label)
		if label == "" {
			label = FreightDescription
		}
		line := &invoice.InvoiceLine{
			ShipmentID:          shp.ID,
			ShipmentProNumber:   shp.ProNumber,
			ShipmentBOL:         shp.BOL,
			LineNumber:          startLineNumber + i,
			Type:                invoice.InvoiceLineTypeFreight,
			Description:         ShareDescription(label, charge),
			Quantity:            decimal.NewFromInt(1),
			UnitPrice:           amount,
			Amount:              amount,
			FormulaTemplateID:   templateID,
			FormulaTemplateName: templateName,
			RateUnit:            freight.RateUnit,
		}
		if freight.Rate != nil {
			line.Rate = decimal.NewNullDecimal(*freight.Rate)
		}
		if quantity, unitPrice, ok := itemized(
			freight.Quantity,
			freight.Rate,
			amount,
			charge.Partial,
		); ok {
			line.Quantity, line.UnitPrice = quantity, unitPrice
		}
		stampShare(line, charge)
		out = append(out, line)
	}

	return out
}

func itemized(
	quantity, rate *decimal.Decimal,
	amount decimal.Decimal,
	partial bool,
) (unitQuantity, unitPrice decimal.Decimal, ok bool) {
	if partial || quantity == nil || !quantity.IsPositive() {
		return decimal.Decimal{}, decimal.Decimal{}, false
	}
	unitPrice = amount.Div(*quantity)
	if rate != nil && rate.Mul(*quantity).Round(chargeallocation.SharePlaces).Equal(amount.Abs()) {
		unitPrice = *rate
		if amount.IsNegative() {
			unitPrice = unitPrice.Neg()
		}
	}
	return *quantity, unitPrice, true
}

func accessorialLine(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	allocated chargeallocation.AllocatedCharge,
	freight decimal.Decimal,
	lineNumber int,
) *invoice.InvoiceLine {
	charge := allocated.Charge
	quantity := decimal.NewFromInt(int64(charge.Unit))
	if quantity.LessThanOrEqual(decimal.Zero) {
		quantity = decimal.NewFromInt(1)
	}

	amount := SignedAmount(billType, allocated.Amount)

	line := &invoice.InvoiceLine{
		ShipmentID:          shp.ID,
		ShipmentProNumber:   shp.ProNumber,
		ShipmentBOL:         shp.BOL,
		LineNumber:          lineNumber,
		Type:                invoice.InvoiceLineTypeAccessorial,
		Description:         AccessorialDescription,
		Quantity:            quantity,
		UnitPrice:           amount.Div(quantity),
		Amount:              amount,
		AccessorialChargeID: charge.AccessorialChargeID,
		AdditionalChargeID:  charge.ID,
		ChargeMethod:        charge.Method,
		Rate:                decimal.NewNullDecimal(charge.Amount),
		RateUnit:            charge.RateUnit,
	}
	if charge.Rate.Valid {
		line.Rate = charge.Rate
	}
	if itemQuantity, unitPrice, ok := itemized(
		nullable(charge.Quantity),
		nullable(charge.Rate),
		amount,
		allocated.Partial,
	); ok {
		line.Quantity, line.UnitPrice = itemQuantity, unitPrice
	}

	if charge.Method == accessorialcharge.MethodPercentage {
		line.RateBasisAmount = decimal.NewNullDecimal(freight)
	}

	if definition := matchingAccessorial(charge); definition != nil {
		line.ChargeCode = strings.TrimSpace(definition.Code)
		if description := strings.TrimSpace(definition.Description); description != "" {
			line.Description = description
		} else if line.ChargeCode != "" {
			line.Description = line.ChargeCode
		}
		if line.RateUnit == "" && charge.Method == accessorialcharge.MethodPerUnit {
			line.RateUnit = definition.RateUnit.TariffRateUnit()
		}
	}
	if label := strings.TrimSpace(charge.Label); label != "" {
		line.Description = label
	}

	line.Description = ShareDescription(line.Description, allocated)
	stampShare(line, allocated)

	return line
}

func nullable(value decimal.NullDecimal) *decimal.Decimal {
	if !value.Valid {
		return nil
	}
	return &value.Decimal
}

func RatingNotes(shipments ...*shipment.Shipment) []string {
	var notes []string
	seen := make(map[string]struct{})
	for _, shp := range shipments {
		if shp == nil || shp.RatingDetail == nil || shp.RatingDetail.Bill == nil {
			continue
		}
		for _, note := range shp.RatingDetail.Bill.Notes {
			text := strings.TrimSpace(note.Text)
			if text == "" {
				continue
			}
			if _, ok := seen[text]; ok {
				continue
			}
			seen[text] = struct{}{}
			notes = append(notes, text)
		}
	}
	return notes
}

func matchingAccessorial(charge *shipment.AdditionalCharge) *accessorialcharge.AccessorialCharge {
	if charge.AccessorialCharge == nil ||
		charge.AccessorialCharge.ID != charge.AccessorialChargeID {
		return nil
	}

	return charge.AccessorialCharge
}

func ratedWith(shp *shipment.Shipment) (id pulid.ID, name string) {
	assignedID, assignedName, assigned := assignedTemplate(shp)
	if shp.RatingDetail == nil {
		return assignedID, assignedName
	}

	name = strings.TrimSpace(shp.RatingDetail.FormulaTemplateName)
	if rated := strings.TrimSpace(shp.RatingDetail.FormulaTemplateID); rated != "" {
		if !pulid.LooksLike(rated) {
			return "", name
		}
		return pulid.ID(rated), name
	}
	if name == "" {
		return assignedID, assignedName
	}
	if assigned && assignedName == name {
		return assignedID, name
	}

	return "", name
}

func assignedTemplate(shp *shipment.Shipment) (id pulid.ID, name string, ok bool) {
	if shp.FormulaTemplate == nil || shp.FormulaTemplate.ID != shp.FormulaTemplateID {
		return "", "", false
	}

	return shp.FormulaTemplate.ID, strings.TrimSpace(shp.FormulaTemplate.Name), true
}

func HydrateAccessorials(
	ctx context.Context,
	repo accessorialchargeports.AccessorialChargeRepository,
	tenantInfo pagination.TenantInfo,
	shipments ...*shipment.Shipment,
) error {
	if repo == nil {
		return nil
	}

	resolved := make(map[pulid.ID]*accessorialcharge.AccessorialCharge)
	for _, shp := range shipments {
		if shp == nil {
			continue
		}
		for _, charge := range shp.AdditionalCharges {
			if charge == nil || charge.AccessorialChargeID.IsNil() ||
				matchingAccessorial(charge) != nil {
				continue
			}

			definition, ok := resolved[charge.AccessorialChargeID]
			if !ok {
				var err error
				definition, err = repo.GetByID(
					ctx,
					accessorialchargeports.GetAccessorialChargeByIDRequest{
						ID:         charge.AccessorialChargeID,
						TenantInfo: &tenantInfo,
					},
				)
				if err != nil {
					return err
				}
				resolved[charge.AccessorialChargeID] = definition
			}
			charge.AccessorialCharge = definition
		}
	}

	return nil
}

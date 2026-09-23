package invoicelines

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	startLineNumber int,
) []*invoice.InvoiceLine {
	return ForShipmentShare(billType, shp, nil, startLineNumber)
}

// ForShipmentShare emits only the lines one payer is billed for. A nil share
// bills the whole shipment, which is what every single-payer invoice is. A
// partial share carries the percent it bills on the line and in its description,
// so the invoice reads correctly on its own.
func ForShipmentShare(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	share *shipment.PayerShare,
	startLineNumber int,
) []*invoice.InvoiceLine {
	if shp == nil {
		return nil
	}
	if share == nil {
		resolution, err := shipment.ResolveShares(shp, nil)
		if err != nil || resolution == nil || len(resolution.Shares) == 0 {
			return nil
		}
		share = resolution.Shares[0]
	}

	lines := make([]*invoice.InvoiceLine, 0, len(share.Charges))
	freight := shp.FreightChargeAmount.Decimal
	lineNumber := startLineNumber

	for _, charge := range share.Charges {
		switch charge.Kind {
		case shipment.ChargeAllocationKindFreight:
			lines = append(lines, freightLine(billType, shp, charge, lineNumber))
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
	charge shipment.AllocatedCharge,
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
func ShareDescription(base string, charge shipment.AllocatedCharge) string {
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

func stampShare(line *invoice.InvoiceLine, charge shipment.AllocatedCharge) {
	if !charge.Partial {
		return
	}
	line.AllocationPercent = charge.Percent
	line.ChargeAllocationID = charge.AllocationID
}

func freightLine(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	charge shipment.AllocatedCharge,
	lineNumber int,
) *invoice.InvoiceLine {
	freightAmount := SignedAmount(billType, charge.Amount)

	line := &invoice.InvoiceLine{
		ShipmentID:          shp.ID,
		ShipmentProNumber:   shp.ProNumber,
		ShipmentBOL:         shp.BOL,
		LineNumber:          lineNumber,
		Type:                invoice.InvoiceLineTypeFreight,
		Description:         ShareDescription(FreightDescription, charge),
		Quantity:            decimal.NewFromInt(1),
		UnitPrice:           freightAmount,
		Amount:              freightAmount,
		FormulaTemplateName: formulaTemplateName(shp),
	}
	if shp.BaseRate.Valid && !shp.BaseRate.Decimal.IsZero() {
		line.Rate = decimal.NewNullDecimal(shp.BaseRate.Decimal)
	}
	stampShare(line, charge)

	return line
}

func accessorialLine(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	allocated shipment.AllocatedCharge,
	freight decimal.Decimal,
	lineNumber int,
) *invoice.InvoiceLine {
	charge := allocated.Charge
	quantity := decimal.NewFromInt(int64(charge.Unit))
	if quantity.LessThanOrEqual(decimal.Zero) {
		quantity = decimal.NewFromInt(1)
	}

	amount := SignedAmount(billType, allocated.Amount)
	unitPrice := amount.Div(quantity)

	line := &invoice.InvoiceLine{
		ShipmentID:          shp.ID,
		ShipmentProNumber:   shp.ProNumber,
		ShipmentBOL:         shp.BOL,
		LineNumber:          lineNumber,
		Type:                invoice.InvoiceLineTypeAccessorial,
		Description:         AccessorialDescription,
		Quantity:            quantity,
		UnitPrice:           unitPrice,
		Amount:              amount,
		AccessorialChargeID: charge.AccessorialChargeID,
		ChargeMethod:        charge.Method,
		Rate:                decimal.NewNullDecimal(charge.Amount),
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
		if charge.Method == accessorialcharge.MethodPerUnit {
			line.RateUnit = definition.RateUnit
		}
	}

	line.Description = ShareDescription(line.Description, allocated)
	stampShare(line, allocated)

	return line
}

func matchingAccessorial(charge *shipment.AdditionalCharge) *accessorialcharge.AccessorialCharge {
	if charge.AccessorialCharge == nil ||
		charge.AccessorialCharge.ID != charge.AccessorialChargeID {
		return nil
	}

	return charge.AccessorialCharge
}

func formulaTemplateName(shp *shipment.Shipment) string {
	if shp.RatingDetail != nil {
		if name := strings.TrimSpace(shp.RatingDetail.FormulaTemplateName); name != "" {
			return name
		}
	}
	if shp.FormulaTemplate != nil && shp.FormulaTemplate.ID == shp.FormulaTemplateID {
		return strings.TrimSpace(shp.FormulaTemplate.Name)
	}

	return ""
}

func HydrateAccessorials(
	ctx context.Context,
	repo repositories.AccessorialChargeRepository,
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
				definition, err = repo.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
					ID:         charge.AccessorialChargeID,
					TenantInfo: &tenantInfo,
				})
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

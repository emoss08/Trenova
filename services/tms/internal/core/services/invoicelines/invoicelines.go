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

func ForShipment(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	startLineNumber int,
) []*invoice.InvoiceLine {
	lines := make([]*invoice.InvoiceLine, 0, 1+len(shp.AdditionalCharges))
	freight := shp.FreightChargeAmount.Decimal
	freightAmount := SignedAmount(billType, freight)

	freightLine := &invoice.InvoiceLine{
		ShipmentID:          shp.ID,
		ShipmentProNumber:   shp.ProNumber,
		ShipmentBOL:         shp.BOL,
		LineNumber:          startLineNumber,
		Type:                invoice.InvoiceLineTypeFreight,
		Description:         FreightDescription,
		Quantity:            decimal.NewFromInt(1),
		UnitPrice:           freightAmount,
		Amount:              freightAmount,
		FormulaTemplateName: formulaTemplateName(shp),
	}
	if shp.BaseRate.Valid && !shp.BaseRate.Decimal.IsZero() {
		freightLine.Rate = decimal.NewNullDecimal(shp.BaseRate.Decimal)
	}
	lines = append(lines, freightLine)

	lineNumber := startLineNumber
	for _, charge := range shp.AdditionalCharges {
		if charge == nil {
			continue
		}
		lineNumber++
		lines = append(lines, accessorialLine(billType, shp, charge, freight, lineNumber))
	}

	return lines
}

func accessorialLine(
	billType billingqueue.BillType,
	shp *shipment.Shipment,
	charge *shipment.AdditionalCharge,
	freight decimal.Decimal,
	lineNumber int,
) *invoice.InvoiceLine {
	quantity := decimal.NewFromInt(int64(charge.Unit))
	if quantity.LessThanOrEqual(decimal.Zero) {
		quantity = decimal.NewFromInt(1)
	}

	amount := SignedAmount(billType, charge.Total(freight))
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

	return line
}

func matchingAccessorial(charge *shipment.AdditionalCharge) *accessorialcharge.AccessorialCharge {
	if charge.AccessorialCharge == nil || charge.AccessorialCharge.ID != charge.AccessorialChargeID {
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

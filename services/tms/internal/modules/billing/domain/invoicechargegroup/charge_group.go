package invoicechargegroup

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ChargeGroup struct {
	Type                invoice.InvoiceLineType
	Description         string
	ChargeCode          string
	ChargeMethod        accessorialcharge.Method
	RateUnit            string
	Rate                decimal.NullDecimal
	RateBasisAmount     decimal.NullDecimal
	Quantity            decimal.Decimal
	FormulaTemplateID   pulid.ID
	FormulaTemplateName string
	LineCount           int
	ShipmentCount       int
	Amount              decimal.Decimal
}

type chargeGroupKey struct {
	lineType            invoice.InvoiceLineType
	accessorialChargeID pulid.ID
	chargeCode          string
	chargeMethod        accessorialcharge.Method
	rateUnit            string
	formulaTemplateID   pulid.ID
	formulaTemplateName string
	description         string
}

type chargeGroupTally struct {
	group     *ChargeGroup
	shipments map[pulid.ID]struct{}
}

func ChargeGroups(lines []*invoice.InvoiceLine) []*ChargeGroup {
	tallies := make(map[chargeGroupKey]*chargeGroupTally, len(lines))
	order := make([]*chargeGroupTally, 0, len(lines))

	for _, line := range lines {
		if line == nil {
			continue
		}

		key := chargeGroupKeyOf(line)
		tally, ok := tallies[key]
		if !ok {
			tally = &chargeGroupTally{
				group: &ChargeGroup{
					Type:                line.Type,
					Description:         line.Description,
					ChargeCode:          line.ChargeCode,
					ChargeMethod:        line.ChargeMethod,
					RateUnit:            line.RateUnit,
					Rate:                line.Rate,
					RateBasisAmount:     line.RateBasisAmount,
					FormulaTemplateID:   line.FormulaTemplateID,
					FormulaTemplateName: line.FormulaTemplateName,
				},
				shipments: make(map[pulid.ID]struct{}, 1),
			}
			tallies[key] = tally
			order = append(order, tally)
		} else {
			tally.add(line)
		}

		tally.group.LineCount++
		tally.group.Quantity = tally.group.Quantity.Add(line.Quantity)
		tally.group.Amount = tally.group.Amount.Add(line.Amount)
		if line.ShipmentID.IsNotNil() {
			tally.shipments[line.ShipmentID] = struct{}{}
		}
	}

	slices.SortStableFunc(order, func(a, b *chargeGroupTally) int {
		return lineTypeRank(a.group.Type) - lineTypeRank(b.group.Type)
	})

	groups := make([]*ChargeGroup, 0, len(order))
	for _, tally := range order {
		tally.group.ShipmentCount = len(tally.shipments)
		groups = append(groups, tally.group)
	}

	return groups
}

func (t *chargeGroupTally) add(line *invoice.InvoiceLine) {
	if !sameDecimal(t.group.Rate, line.Rate) {
		t.group.Rate = decimal.NullDecimal{}
	}
	t.group.RateBasisAmount = decimal.NullDecimal{}
}

func chargeGroupKeyOf(line *invoice.InvoiceLine) chargeGroupKey {
	key := chargeGroupKey{
		lineType:            line.Type,
		accessorialChargeID: line.AccessorialChargeID,
		chargeCode:          line.ChargeCode,
		chargeMethod:        line.ChargeMethod,
		rateUnit:            line.RateUnit,
		formulaTemplateID:   line.FormulaTemplateID,
		formulaTemplateName: line.FormulaTemplateName,
	}
	if line.Type == invoice.InvoiceLineTypeFreight || line.AccessorialChargeID.IsNil() {
		key.description = line.Description
	}

	return key
}

func sameDecimal(a, b decimal.NullDecimal) bool {
	if !a.Valid || !b.Valid {
		return a.Valid == b.Valid
	}

	return a.Decimal.Equal(b.Decimal)
}

func lineTypeRank(lineType invoice.InvoiceLineType) int {
	switch lineType {
	case invoice.InvoiceLineTypeFreight:
		return 0
	case invoice.InvoiceLineTypeAccessorial:
		return 1
	case invoice.InvoiceLineTypeMemo:
		return 2
	default:
		return 3
	}
}

func BilledShipmentIDs(invoices ...*invoice.Invoice) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(invoices))
	ids := make([]pulid.ID, 0, len(invoices))
	keep := func(id pulid.ID) {
		if id.IsNil() {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	for _, inv := range invoices {
		if inv == nil {
			continue
		}
		keep(inv.ShipmentID)
		for _, line := range inv.Lines {
			if line != nil {
				keep(line.ShipmentID)
			}
		}
	}

	return ids
}

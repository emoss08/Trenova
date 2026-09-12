package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

// loadInvoiceLegs resolves and fetches every shipment billed by an invoice — one leg
// for a single-shipment invoice, the set of line-attributed legs for a grouped (order)
// invoice.
func loadInvoiceLegs(
	ctx context.Context,
	shipmentRepo repositories.ShipmentRepository,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) ([]*shipment.Shipment, error) {
	legIDs := entity.LegShipmentIDs()
	if len(legIDs) == 0 {
		return nil, nil
	}

	legs := make([]*shipment.Shipment, 0, len(legIDs))
	for _, legID := range legIDs {
		shp, err := shipmentRepo.GetByID(ctx, basicShipmentByIDRequest(legID, tenantInfo))
		if err != nil {
			return nil, err
		}
		legs = append(legs, shp)
	}

	return legs, nil
}

func reconciliationRelatedEntities(entity *invoice.Invoice) map[string]any {
	related := map[string]any{"invoiceId": entity.ID.String()}
	if !entity.ShipmentID.IsNil() {
		related["shipmentId"] = entity.ShipmentID.String()
	}
	if !entity.OrderID.IsNil() {
		related["orderId"] = entity.OrderID.String()
	}
	return related
}

// reconciliationExpectedTotal is the source-of-truth amount an invoice should
// reconcile against: the signed sum of the current rated totals of its legs, plus —
// for anything wider than one shipment — the unattributed lines at their invoiced
// amount. Only a single-shipment invoice has no unattributed lines to add; keying
// this off a nil order id instead dropped them from every consolidated invoice.
func reconciliationExpectedTotal(
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
) decimal.Decimal {
	legTotal := decimal.Zero
	for _, shp := range legs {
		legTotal = legTotal.Add(shp.TotalChargeAmount.Decimal)
	}

	expected := signedAmount(entity.BillType, legTotal)
	if entity.Scope == invoice.ScopeShipment {
		return expected
	}

	for _, line := range entity.Lines {
		if line == nil || !line.ShipmentID.IsNil() {
			continue
		}
		expected = expected.Add(line.Amount)
	}

	return expected
}

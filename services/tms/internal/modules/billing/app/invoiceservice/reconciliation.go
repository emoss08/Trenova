package invoiceservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicelines"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

// loadInvoiceLegs resolves and fetches every shipment billed by an invoice — one leg
// for a single-shipment invoice, the set of line-attributed legs for a grouped (order)
// invoice.
func loadInvoiceLegs(
	ctx context.Context,
	shipmentRepo shipmentports.ShipmentRepository,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	withDetails bool,
) ([]*shipment.Shipment, error) {
	legIDs := entity.LegShipmentIDs()
	if len(legIDs) == 0 {
		return nil, nil
	}

	legs := make([]*shipment.Shipment, 0, len(legIDs))
	for _, legID := range legIDs {
		shp, err := shipmentRepo.GetByID(ctx, shipmentByIDRequest(legID, tenantInfo, withDetails))
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
	ctx context.Context,
	entity *invoice.Invoice,
	legs []*shipment.Shipment,
) (decimal.Decimal, error) {
	legTotal := decimal.Zero
	for _, shp := range legs {
		total, err := reconciliationLegTotal(ctx, entity, shp)
		if err != nil {
			return decimal.Zero, err
		}
		legTotal = legTotal.Add(total)
	}

	expected := invoicelines.SignedAmount(entity.BillType, legTotal)
	if entity.Scope == invoice.ScopeShipment {
		return expected, nil
	}

	for _, line := range entity.Lines {
		if line == nil || !line.ShipmentID.IsNil() {
			continue
		}
		expected = expected.Add(line.Amount)
	}

	return expected, nil
}

// reconciliationLegTotal is what one leg should have contributed: the whole
// shipment on an ordinary invoice, the payer's share on a split one. Legs are
// loaded with their allocations, so no second read is needed.
func reconciliationLegTotal(
	ctx context.Context,
	entity *invoice.Invoice,
	shp *shipment.Shipment,
) (decimal.Decimal, error) {
	if shp == nil {
		return decimal.Zero, nil
	}
	if !entity.IsSplitBill {
		return shp.TotalChargeAmount.Decimal, nil
	}

	resolution, err := chargeallocation.ResolveShares(ctx, shp, shp.ChargeAllocations)
	var refused *errortypes.Error
	switch {
	case errors.As(err, &refused):
		return shp.TotalChargeAmount.Decimal, nil
	case err != nil:
		return decimal.Zero, err
	}
	if share := resolution.ShareFor(entity.CustomerID); share != nil {
		return share.TotalAmount, nil
	}

	return decimal.Zero, nil
}

package invoicevoid

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Deps is what releasing a voided invoice's freight needs. It is a bundle of
// repositories rather than services so that both the invoice service and the
// adjustment engine can call it without depending on each other.
type Deps struct {
	BillingQueueRepo     repositories.BillingQueueRepository
	OrderRepo            repositories.OrderRepository
	ChargeAllocationRepo repositories.ChargeAllocationRepository
	ShipmentRepo         repositories.ShipmentRepository
	InvoiceRepo          repositories.InvoiceRepository
	OrderDerivation      servicesports.OrderDerivationService
	Renumber             func(ctx context.Context, billType billingqueue.BillType) (string, error)
}

// Params names the invoice being voided and how its freight should be treated.
type Params struct {
	Invoice     *invoice.Invoice
	Disposition invoice.VoidDisposition
	ActorUserID pulid.ID
	Reason      string
	// WasPosted says whether the invoice had flipped its shipments to Invoiced,
	// which is what decides whether they need putting back.
	WasPosted bool
	Now       int64
}

// Release hands back everything a voided invoice held: its queue items, the
// order charges and allocation shares it carried, and the shipments it marked
// invoiced. Rebill leaves the freight ready to bill again; DoNotRebill retires
// it. It runs inside the caller's transaction.
func Release(ctx context.Context, deps Deps, p Params) ([]pulid.ID, error) {
	if p.Invoice == nil {
		return nil, nil
	}
	entity := p.Invoice
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	rebill := p.Disposition == invoice.VoidDispositionRebill

	released, err := deps.BillingQueueRepo.ReleaseForInvoice(
		ctx,
		&repositories.ReleaseForInvoiceRequest{
			TenantInfo:   tenantInfo,
			InvoiceID:    entity.ID,
			AnchorItemID: entity.BillingQueueItemID,
			Rebill:       rebill,
			CanceledByID: pulid.PtrOrNil(p.ActorUserID),
			CanceledAt:   p.Now,
			CancelReason: cancelReason(p.Reason),
			RenumberFn:   deps.Renumber,
		},
	)
	if err != nil {
		return nil, err
	}
	releasedIDs := make([]pulid.ID, 0, len(released))
	for _, item := range released {
		if item != nil {
			releasedIDs = append(releasedIDs, item.ID)
		}
	}

	if deps.OrderRepo != nil {
		if _, err = deps.OrderRepo.ClearChargesInvoice(ctx, &repositories.ClearOrderChargesInvoiceRequest{
			TenantInfo: tenantInfo,
			InvoiceID:  entity.ID,
		}); err != nil {
			return nil, err
		}
	}
	if deps.ChargeAllocationRepo != nil {
		if _, err = deps.ChargeAllocationRepo.ClearInvoice(ctx, &repositories.ClearChargeAllocationsInvoiceRequest{
			TenantInfo: tenantInfo,
			InvoiceID:  entity.ID,
		}); err != nil {
			return nil, err
		}
	}

	if !p.WasPosted || deps.ShipmentRepo == nil {
		return releasedIDs, nil
	}

	if err = restoreShipments(ctx, deps, entity, tenantInfo, rebill); err != nil {
		return nil, err
	}

	return releasedIDs, nil
}

// restoreShipments puts the invoice's legs back where they were before it
// posted. A leg another payer's posted invoice still bills stays Invoiced under
// DoNotRebill: that other invoice is still good.
func restoreShipments(
	ctx context.Context,
	deps Deps,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	rebill bool,
) error {
	legIDs := entity.LegShipmentIDs()
	if len(legIDs) == 0 {
		return nil
	}

	otherInvoices := map[pulid.ID][]*invoice.Invoice{}
	if deps.InvoiceRepo != nil {
		var err error
		otherInvoices, err = deps.InvoiceRepo.ListByShipmentIDs(
			ctx,
			repositories.ListInvoicesByShipmentIDsRequest{
				TenantInfo:  tenantInfo,
				ShipmentIDs: legIDs,
			},
		)
		if err != nil {
			return err
		}
	}

	orderIDs := make(map[pulid.ID]struct{}, len(legIDs))
	for _, legID := range legIDs {
		shp, err := deps.ShipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         legID,
			TenantInfo: tenantInfo,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		})
		if err != nil {
			return err
		}

		stillBilled := false
		for _, other := range otherInvoices[legID] {
			if other == nil || other.ID == entity.ID || other.Status != invoice.StatusPosted ||
				other.BillType == billingqueue.BillTypeCreditMemo {
				continue
			}
			stillBilled = true
			break
		}

		switch {
		case rebill:
			shp.Status = shipment.StatusReadyToInvoice
			shp.BilledAt = nil
		case stillBilled:
			shp.BilledAt = nil
		default:
			shp.Status = shipment.StatusCompleted
			shp.BilledAt = nil
		}
		if _, err = deps.ShipmentRepo.UpdateDerivedState(ctx, shp); err != nil {
			return err
		}
		if !shp.OrderID.IsNil() {
			orderIDs[shp.OrderID] = struct{}{}
		}
	}
	if !entity.OrderID.IsNil() {
		orderIDs[entity.OrderID] = struct{}{}
	}

	if deps.OrderDerivation == nil {
		return nil
	}
	for orderID := range orderIDs {
		if err := deps.OrderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
			return err
		}
	}

	return nil
}

func cancelReason(reason string) string {
	const limit = 100
	prefix := "Invoice voided: "
	if len(prefix)+len(reason) <= limit {
		return prefix + reason
	}
	if len(prefix) >= limit {
		return prefix[:limit]
	}

	return prefix + reason[:limit-len(prefix)]
}

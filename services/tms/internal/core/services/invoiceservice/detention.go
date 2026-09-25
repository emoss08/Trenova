package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// guardDetentionHolds refuses to bill shipments while a detention charge on
// any of them is waiting on an approver.
func (s *Service) guardDetentionHolds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) error {
	if s.detentionBilling == nil || len(shipmentIDs) == 0 {
		return nil
	}

	return s.detentionBilling.GuardShipments(ctx, &servicesports.DetentionBillingHoldsRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: shipmentIDs,
	})
}

// guardItemDetentionHolds is the gate for one queue item. A credit memo gives
// money back and bills no charge, so nothing holds it.
func (s *Service) guardItemDetentionHolds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *billingqueue.BillingQueueItem,
) error {
	if item == nil || item.ShipmentID.IsNil() || item.BillType == billingqueue.BillTypeCreditMemo {
		return nil
	}

	return s.guardDetentionHolds(ctx, tenantInfo, []pulid.ID{item.ShipmentID})
}

// syncDetentionBilling marks billed the detention charges a new invoice
// carries. It runs in the transaction that created the invoice.
func (s *Service) syncDetentionBilling(
	ctx context.Context,
	created *invoice.Invoice,
	actor *servicesports.RequestActor,
) error {
	if s.detentionBilling == nil || created == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: created.OrganizationID,
		BuID:  created.BusinessUnitID,
	}

	return s.detentionBilling.SyncInvoiceBilling(
		ctx,
		&servicesports.SyncDetentionInvoiceBillingRequest{
			TenantInfo:    tenantInfo,
			InvoiceIDs:    []pulid.ID{created.ID},
			InvoiceNumber: created.Number,
			Event:         servicesports.DetentionBillingInvoiceCreated,
			ActorUserID:   actorUserID(actor, tenantInfo),
		},
	)
}

func legIDs(legs []*shipment.Shipment) []pulid.ID {
	ids := make([]pulid.ID, 0, len(legs))
	for _, leg := range legs {
		if leg != nil && leg.ID.IsNotNil() {
			ids = append(ids, leg.ID)
		}
	}

	return ids
}

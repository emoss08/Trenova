package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
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

	return s.detentionBilling.GuardShipments(ctx, &detentionports.DetentionBillingHoldsRequest{
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

func guardItemRatingReview(item *billingqueue.BillingQueueItem, entity *shipment.Shipment) error {
	if item == nil || item.BillType == billingqueue.BillTypeCreditMemo || entity == nil {
		return nil
	}

	return shipment.RatingReviewHoldError([]*shipment.Shipment{entity})
}

// syncDetentionBilling marks billed the detention charges a new invoice
// carries. It runs in the transaction that created the invoice.
func (s *Service) syncDetentionBilling(
	ctx context.Context,
	created *invoice.Invoice,
	actor *permissionports.RequestActor,
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
		&detentionports.SyncDetentionInvoiceBillingRequest{
			TenantInfo:    tenantInfo,
			InvoiceIDs:    []pulid.ID{created.ID},
			InvoiceNumber: created.Number,
			Event:         detentionports.DetentionBillingInvoiceCreated,
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

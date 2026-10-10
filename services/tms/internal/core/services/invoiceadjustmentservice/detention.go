package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// guardDetentionRebill refuses a replacement invoice for shipments with a
// detention charge still waiting on an approver, the same gate every other
// invoice passes. A refusal becomes a preview error, so the person adjusting
// sees it before anything is written.
func (s *Service) guardDetentionRebill(
	ctx context.Context,
	preview *invoiceadjustmentports.InvoiceAdjustmentPreview,
	tenantInfo pagination.TenantInfo,
	source *invoice.Invoice,
) error {
	if s.detentionBilling == nil || source == nil {
		return nil
	}

	shipmentIDs := source.LegShipmentIDs()
	if len(shipmentIDs) == 0 {
		return nil
	}

	err := s.detentionBilling.GuardShipments(ctx, &detentionports.DetentionBillingHoldsRequest{
		TenantInfo:  tenantInfo,
		ShipmentIDs: shipmentIDs,
	})
	if err == nil {
		return nil
	}
	if errortypes.IsBusinessError(err) {
		appendPreviewError(preview, "lines", err.Error())
		return nil
	}

	return err
}

// syncDetentionBilling settles the detention charges an adjustment touched:
// those the original invoice billed, and any the replacement bills.
func (s *Service) syncDetentionBilling(
	ctx context.Context,
	source *invoice.Invoice,
	actor *permissionports.RequestActor,
	documents ...*invoice.Invoice,
) error {
	if s.detentionBilling == nil || source == nil {
		return nil
	}

	invoiceIDs := make([]pulid.ID, 0, len(documents)+1)
	invoiceIDs = append(invoiceIDs, source.ID)
	for _, document := range documents {
		if document != nil {
			invoiceIDs = append(invoiceIDs, document.ID)
		}
	}

	var actorUserID pulid.ID
	if actor != nil {
		actorUserID = actor.UserID
	}

	return s.detentionBilling.SyncInvoiceBilling(
		ctx,
		&detentionports.SyncDetentionInvoiceBillingRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: source.OrganizationID,
				BuID:  source.BusinessUnitID,
			},
			InvoiceIDs:    invoiceIDs,
			InvoiceNumber: source.Number,
			Event:         detentionports.DetentionBillingInvoiceAdjusted,
			ActorUserID:   actorUserID,
		},
	)
}

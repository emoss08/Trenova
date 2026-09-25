package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
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
	preview *servicesports.InvoiceAdjustmentPreview,
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

	err := s.detentionBilling.GuardShipments(ctx, &servicesports.DetentionBillingHoldsRequest{
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
	actor *servicesports.RequestActor,
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
		&servicesports.SyncDetentionInvoiceBillingRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: source.OrganizationID,
				BuID:  source.BusinessUnitID,
			},
			InvoiceIDs:    invoiceIDs,
			InvoiceNumber: source.Number,
			Event:         servicesports.DetentionBillingInvoiceAdjusted,
			ActorUserID:   actorUserID,
		},
	)
}

package invoiceadjustmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
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

type detentionSync struct {
	source    *invoice.Invoice
	actor     *permissionports.RequestActor
	waivers   []detentionports.DetentionWaiver
	documents []*invoice.Invoice
}

func (s *Service) syncDetentionBilling(ctx context.Context, p *detentionSync) error {
	if s.detentionBilling == nil || p.source == nil {
		return nil
	}

	invoiceIDs := make([]pulid.ID, 0, len(p.documents)+1)
	invoiceIDs = append(invoiceIDs, p.source.ID)
	for _, document := range p.documents {
		if document != nil {
			invoiceIDs = append(invoiceIDs, document.ID)
		}
	}

	var actorUserID pulid.ID
	if p.actor != nil {
		actorUserID = p.actor.UserID
	}

	return s.detentionBilling.SyncInvoiceBilling(
		ctx,
		&detentionports.SyncDetentionInvoiceBillingRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: p.source.OrganizationID,
				BuID:  p.source.BusinessUnitID,
			},
			InvoiceIDs:    invoiceIDs,
			InvoiceNumber: p.source.Number,
			Event:         detentionports.DetentionBillingInvoiceAdjusted,
			ActorUserID:   actorUserID,
			Waivers:       p.waivers,
		},
	)
}

func adjustmentMetadata(req *invoiceadjustmentports.InvoiceAdjustmentRequest) map[string]any {
	metadata := make(map[string]any, 2)
	metadata[metadataAttachmentIDs] = req.AttachmentIDs
	if len(req.DetentionWaivers) > 0 {
		metadata[metadataDetentionWaivers] = req.DetentionWaivers
	}

	return metadata
}

func detentionWaiversOf(
	adjustment *invoiceadjustment.InvoiceAdjustment,
) ([]detentionports.DetentionWaiver, error) {
	raw, ok := adjustment.Metadata[metadataDetentionWaivers]
	if !ok || raw == nil {
		return nil, nil
	}

	var waivers []detentionports.DetentionWaiver
	if err := jsonutils.Convert(raw, &waivers); err != nil {
		return nil, fmt.Errorf(
			"read the detention waivers of adjustment %s: %w",
			adjustment.ID,
			err,
		)
	}

	return waivers, nil
}

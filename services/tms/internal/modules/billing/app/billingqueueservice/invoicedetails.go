package billingqueueservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	maxPONumberLength   = 100
	maxInvoiceMemoChars = 2000
	maxPacketDocuments  = 25
)

func (s *Service) UpdateInvoiceDetails(
	ctx context.Context,
	req *billingqueueports.UpdateInvoiceDetailsRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	item, err := s.repo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if billingqueue.IsTerminalStatus(item.Status) || item.Status == billingqueue.StatusApproved {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"The invoice is already made; change it on the invoice instead",
		)
	}

	packet := sliceutils.Dedupe(req.PacketDocumentIDs)
	if multiErr := validateInvoiceDetails(req, packet); multiErr != nil {
		return nil, multiErr
	}
	if err = s.ensurePacketDocuments(ctx, item, packet); err != nil {
		return nil, err
	}

	previous := *item
	item.PONumber = strings.TrimSpace(req.PONumber)
	item.InvoiceMemo = strings.TrimSpace(req.InvoiceMemo)
	item.PaymentTerm = req.PaymentTerm
	item.InvoiceDate = req.InvoiceDate
	item.PacketDocumentIDs = packet
	if req.Version > 0 {
		item.Version = req.Version
	}

	updated, err := s.repo.UpdateInvoiceDetails(ctx, item)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(
		updated,
		auditActor,
		permission.OpUpdate,
		&previous,
		updated,
		"Invoice details updated",
	)
	s.activity.PublishInvalidation(ctx, updated, auditActor, "updated", updated)
	s.syncReview(ctx, updated.ID, req.TenantInfo)

	return s.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                updated.ID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: true,
	})
}

func validateInvoiceDetails(
	req *billingqueueports.UpdateInvoiceDetailsRequest,
	packet []pulid.ID,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if len(strings.TrimSpace(req.PONumber)) > maxPONumberLength {
		multiErr.Add(
			"poNumber",
			errortypes.ErrInvalid,
			"Keep the PO number under {0} characters",
			maxPONumberLength,
		)
	}
	if len(strings.TrimSpace(req.InvoiceMemo)) > maxInvoiceMemoChars {
		multiErr.Add(
			"invoiceMemo",
			errortypes.ErrInvalid,
			"Keep the note under {0} characters",
			maxInvoiceMemoChars,
		)
	}
	if req.PaymentTerm != "" && !invoice.PaymentTerm(req.PaymentTerm).IsValid() {
		multiErr.Add("paymentTerm", errortypes.ErrInvalid, "Choose one of the payment terms")
	}
	if req.InvoiceDate != nil && *req.InvoiceDate <= 0 {
		multiErr.Add("invoiceDate", errortypes.ErrInvalid, "Choose a valid invoice date")
	}
	if len(packet) > maxPacketDocuments {
		multiErr.Add(
			"packetDocumentIds",
			errortypes.ErrInvalid,
			"An invoice packet holds at most {0} documents",
			maxPacketDocuments,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) ensurePacketDocuments(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	packet []pulid.ID,
) error {
	if len(packet) == 0 {
		return nil
	}
	if s.documentRepo == nil || item.ShipmentID.IsNil() {
		return errortypes.NewValidationError(
			"packetDocumentIds",
			errortypes.ErrInvalid,
			"This item has no shipment documents to send",
		)
	}

	docs, err := s.documentRepo.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: item.OrganizationID,
			BuID:  item.BusinessUnitID,
		},
		ResourceID:   item.ShipmentID.String(),
		ResourceType: shipmentResourceType,
	})
	if err != nil {
		return err
	}

	current := make(map[pulid.ID]struct{}, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.IsCurrentVersion && doc.Status != document.StatusRejected &&
			doc.Status != document.StatusArchived {
			current[doc.ID] = struct{}{}
		}
	}
	for _, id := range packet {
		if _, ok := current[id]; !ok {
			return errortypes.NewValidationError(
				"packetDocumentIds",
				errortypes.ErrInvalid,
				"Only the shipment's current documents can go in the invoice packet",
			)
		}
	}

	return nil
}

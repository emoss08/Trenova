package billingqueueservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
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
	maxPONumberLength    = 100
	maxInvoiceMemoChars  = 2000
	maxPacketDocuments   = 25
	maxPacketRotations   = 500
	shipmentResourceType = "shipment"
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
	if packet, err = s.ensurePacketDocuments(ctx, item, packet); err != nil {
		return nil, err
	}
	rotations, err := packetRotations(req.PacketPageRotations, packet)
	if err != nil {
		return nil, err
	}

	previous := *item
	item.PONumber = strings.TrimSpace(req.PONumber)
	item.InvoiceMemo = strings.TrimSpace(req.InvoiceMemo)
	item.PaymentTerm = req.PaymentTerm
	item.InvoiceDate = req.InvoiceDate
	item.PickPacket(packet, req.PacketPicked)
	item.PacketPageRotations = rotations
	if req.Version > 0 {
		item.Version = req.Version
	}

	updated, err := s.repo.UpdateInvoiceDetails(ctx, item)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(ctx,
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
) ([]pulid.ID, error) {
	if len(packet) == 0 {
		return packet, nil
	}
	if item.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
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
		return nil, err
	}

	return carriedPacket(packet, item.PacketDocumentIDs, docs)
}

func carriedPacket(
	packet, carried []pulid.ID,
	docs []*document.Document,
) ([]pulid.ID, error) {
	current := make(map[pulid.ID]struct{}, len(docs))
	for _, doc := range docs {
		if doc != nil && doc.IsCurrentVersion && doc.Status != document.StatusRejected &&
			doc.Status != document.StatusArchived {
			current[doc.ID] = struct{}{}
		}
	}
	kept := make([]pulid.ID, 0, len(packet))
	for _, id := range packet {
		if _, ok := current[id]; ok {
			kept = append(kept, id)
			continue
		}
		if slices.Contains(carried, id) {
			continue
		}
		return nil, errortypes.NewValidationError(
			"packetDocumentIds",
			errortypes.ErrInvalid,
			"Only the shipment's current documents can go in the invoice packet",
		)
	}

	return kept, nil
}

func packetRotations(
	turns []billingqueue.PacketPageRotation,
	packet []pulid.ID,
) ([]billingqueue.PacketPageRotation, error) {
	if len(turns) > maxPacketRotations {
		return nil, errortypes.NewValidationError(
			"packetPageRotations",
			errortypes.ErrInvalid,
			"A packet keeps at most {0} turned pages",
			maxPacketRotations,
		)
	}
	order := make(map[pulid.ID]int, len(packet))
	for idx, id := range packet {
		order[id] = idx
	}
	latest := make(map[billingqueue.PacketPageRotation]int, len(turns))
	for _, turn := range turns {
		rotation, ok := billingvocab.QuarterTurn(turn.Rotation)
		if !ok {
			return nil, errortypes.NewValidationError(
				"packetPageRotations",
				errortypes.ErrInvalid,
				"Pages turn in quarter turns",
			)
		}
		if turn.Page < 1 || turn.Page > billingvocab.MaxRotatedPage {
			return nil, errortypes.NewValidationError(
				"packetPageRotations",
				errortypes.ErrInvalid,
				"Choose a page from 1 to {0}",
				billingvocab.MaxRotatedPage,
			)
		}
		if _, carried := order[turn.DocumentID]; !carried {
			continue
		}
		key := billingqueue.PacketPageRotation{DocumentID: turn.DocumentID, Page: turn.Page}
		latest[key] = rotation
	}

	out := make([]billingqueue.PacketPageRotation, 0, len(latest))
	for key, rotation := range latest {
		if rotation == 0 {
			continue
		}
		out = append(out, billingqueue.PacketPageRotation{
			DocumentID: key.DocumentID,
			Page:       key.Page,
			Rotation:   rotation,
		})
	}
	slices.SortFunc(out, func(a, b billingqueue.PacketPageRotation) int {
		if byDocument := order[a.DocumentID] - order[b.DocumentID]; byDocument != 0 {
			return byDocument
		}
		return a.Page - b.Page
	})

	return out, nil
}

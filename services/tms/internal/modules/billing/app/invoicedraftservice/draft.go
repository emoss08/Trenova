package invoicedraftservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) UpdateDraft(
	ctx context.Context,
	req *invoiceports.UpdateInvoiceDraftRequest,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseDeliveryUpdate(entity, req); err != nil {
		return nil, err
	}

	previous := *entity
	applyDraftUpdate(entity, req)

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	if req.AttachmentIDs != nil {
		if _, err = s.repo.UpsertAttachments(ctx, invoiceports.UpsertInvoiceAttachmentsRequest{
			InvoiceID:      updated.ID,
			DocumentIDs:    *req.AttachmentIDs,
			OrganizationID: updated.OrganizationID,
			BusinessUnitID: updated.BusinessUnitID,
			TenantInfo:     req.TenantInfo,
		}); err != nil {
			return nil, err
		}
	}

	s.rec.LogAction(
		updated,
		actor.AuditActor(),
		permission.OpUpdate,
		&previous,
		updated,
		"Invoice draft delivery metadata updated",
	)
	return s.repo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{ID: updated.ID, TenantInfo: req.TenantInfo},
	)
}

func refuseDeliveryUpdate(
	entity *invoice.Invoice,
	req *invoiceports.UpdateInvoiceDraftRequest,
) error {
	switch entity.Status { //nolint:exhaustive // every status that is neither draft nor posted is refused
	case invoice.StatusDraft:
		return nil
	case invoice.StatusPosted:
		if req.Memo != nil || req.RemittanceInstructions != nil {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"The memo and remittance instructions print on the posted invoice and can only be changed on a draft",
			)
		}
		return nil
	default:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A voided invoice cannot be changed",
		)
	}
}

func applyDraftUpdate(entity *invoice.Invoice, req *invoiceports.UpdateInvoiceDraftRequest) {
	if req.Memo != nil {
		entity.Memo = strings.TrimSpace(*req.Memo)
	}
	if req.RemittanceInstructions != nil {
		entity.RemittanceInstructions = strings.TrimSpace(*req.RemittanceInstructions)
	}
	if req.EmailSubject != nil {
		entity.EmailSubjectSnapshot = strings.TrimSpace(*req.EmailSubject)
	}
	if req.EmailBody != nil {
		entity.EmailBodySnapshot = strings.TrimSpace(*req.EmailBody)
	}
	if req.EmailTo != nil {
		entity.EmailToSnapshot = mailtext.NormalizeEmailList(*req.EmailTo)
	}
	if req.EmailCC != nil {
		entity.EmailCCSnapshot = mailtext.NormalizeEmailList(*req.EmailCC)
	}
	if req.EmailBCC != nil {
		entity.EmailBCCSnapshot = mailtext.NormalizeEmailList(*req.EmailBCC)
	}
}

func (s *Service) PreviewUpdateDraft(
	ctx context.Context,
	req *invoiceports.UpdateInvoiceDraftRequest,
) (*invoiceports.InvoiceDraftUpdatePreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseDeliveryUpdate(entity, req); err != nil {
		return nil, err
	}

	after := *entity
	applyDraftUpdate(&after, req)
	preview := &invoiceports.InvoiceDraftUpdatePreview{
		Before:            entity,
		After:             &after,
		AttachmentsBefore: attachmentDocumentIDs(entity.Attachments),
	}
	preview.AttachmentsAfter = preview.AttachmentsBefore
	if req.AttachmentIDs != nil {
		preview.AttachmentsAfter = *req.AttachmentIDs
	}

	return preview, nil
}

func attachmentDocumentIDs(attachments []*invoice.Attachment) []pulid.ID {
	ids := make([]pulid.ID, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment != nil {
			ids = append(ids, attachment.DocumentID)
		}
	}

	return ids
}

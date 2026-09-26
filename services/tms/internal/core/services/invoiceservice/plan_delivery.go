package invoiceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func refuseNonDraftUpdate(entity *invoice.Invoice) error {
	if entity.Status == invoice.StatusDraft {
		return nil
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalid,
		"Only draft invoices can be updated",
	)
}

func applyDraftUpdate(entity *invoice.Invoice, req *servicesports.UpdateInvoiceDraftRequest) {
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
		entity.EmailToSnapshot = normalizeRecipients(*req.EmailTo)
	}
	if req.EmailCC != nil {
		entity.EmailCCSnapshot = normalizeRecipients(*req.EmailCC)
	}
	if req.EmailBCC != nil {
		entity.EmailBCCSnapshot = normalizeRecipients(*req.EmailBCC)
	}
}

func (s *Service) PreviewUpdateDraft(
	ctx context.Context,
	req *servicesports.UpdateInvoiceDraftRequest,
) (*servicesports.InvoiceDraftUpdatePreview, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseNonDraftUpdate(entity); err != nil {
		return nil, err
	}

	after := *entity
	applyDraftUpdate(&after, req)
	preview := &servicesports.InvoiceDraftUpdatePreview{
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

func (s *Service) planPDFGeneration(
	ctx context.Context,
	req *servicesports.InvoicePreviewRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, pulid.ID, error) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, pulid.Nil, errortypes.NewBusinessError(
			"Invoice PDF generation requires workflow processing to be enabled",
		).WithInternal(servicesports.ErrWorkflowStarterDisabled)
	}
	if req == nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	userID := actorUserID(actor, req.TenantInfo)
	if userID.IsNil() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"userId",
			errortypes.ErrRequired,
			"User ID is required to generate invoice PDFs",
		)
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, pulid.Nil, err
	}
	if err = refuseVoidedInvoice(entity, "rendered"); err != nil {
		return nil, pulid.Nil, err
	}

	return entity, userID, nil
}

func autoSendsOnGeneration(cus *customer.Customer, entity *invoice.Invoice) bool {
	if cus == nil || cus.BillingProfile == nil ||
		!cus.BillingProfile.AutoSendInvoiceOnGeneration ||
		!cus.BillingProfile.EmailInvoiceEnabled {
		return false
	}

	return entity.SendStatus == invoice.SendStatusNotSent ||
		entity.SendStatus == invoice.SendStatusFailed
}

func (s *Service) PlanPDFGeneration(
	ctx context.Context,
	req *servicesports.InvoicePreviewRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoicePDFGenerationPlan, error) {
	entity, _, err := s.planPDFGeneration(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         entity.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &servicesports.InvoicePDFGenerationPlan{
		Invoice:   entity,
		AutoSends: autoSendsOnGeneration(cus, entity),
	}, nil
}

func (s *Service) planEDISend(
	ctx context.Context,
	req *servicesports.SendInvoiceEDIRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, *servicesports.InvoiceEDISendPlan, error) {
	if req == nil {
		return nil, nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if entity.Status != invoice.StatusPosted {
		return nil, nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only a posted invoice can be sent by EDI",
		)
	}
	plans, err := s.ResolveEDISendPlans(ctx, &servicesports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: req.TenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		return nil, nil, err
	}
	plan := plans[entity.ID]
	if plan == nil || !plan.Enabled || len(plan.Blockers) > 0 {
		blocker := ediBlockerProfileDisabled
		if plan != nil && len(plan.Blockers) > 0 {
			blocker = plan.Blockers[0]
		}
		return nil, nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			blocker,
		)
	}
	if !req.Force {
		switch entity.EDISendStatus {
		case invoice.EDISendStatusQueued, invoice.EDISendStatusSending:
			return nil, nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"An EDI send for this invoice is already in progress",
			)
		case invoice.EDISendStatusSent:
			return nil, nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"This invoice has already been sent by EDI; resend it with force",
			)
		}
	}

	return entity, plan, nil
}

func (s *Service) PreviewSendEDI(
	ctx context.Context,
	req *servicesports.SendInvoiceEDIRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceEDISendPreview, error) {
	entity, plan, err := s.planEDISend(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Invoice EDI delivery requires workflow processing to be enabled",
		).WithInternal(servicesports.ErrWorkflowStarterDisabled)
	}

	return &servicesports.InvoiceEDISendPreview{Invoice: entity, Plan: plan}, nil
}

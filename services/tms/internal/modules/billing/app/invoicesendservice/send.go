package invoicesendservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type sendBasis struct {
	profile *invoicedocument.DeliveryProfile
	message string
}

func (s *Service) PlanSend(
	ctx context.Context,
	req *invoiceports.InvoiceSendPlanRequest,
) (*invoiceports.InvoiceSendPlan, error) {
	plan, _, err := s.planSend(ctx, req)
	return plan, err
}

func (s *Service) planSend(
	ctx context.Context,
	req *invoiceports.InvoiceSendPlanRequest,
) (*invoiceports.InvoiceSendPlan, *sendBasis, error) {
	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if err = invoicevalidation.RefuseVoided(entity, "sent"); err != nil {
		return nil, nil, err
	}
	deliveryProfile, err := s.resolveDeliveryProfile(ctx, invoicedocument.DeliveryProfileParams{
		Entity:                        entity,
		TenantInfo:                    req.TenantInfo,
		IncludeShipmentDetails:        true,
		IncludeCustomer:               true,
		IncludeCustomerEmailProfile:   true,
		IncludeCustomerBillingProfile: true,
	})
	if err != nil {
		return nil, nil, err
	}
	var profile *email.Profile
	profile, err = s.emailRepo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeBilling)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, nil, err
	}
	recipients := invoicedocument.ResolveRecipients(entity, deliveryProfile.Email)
	if err = s.resolveDeliveryOrganization(ctx, deliveryProfile, req.TenantInfo); err != nil {
		return nil, nil, err
	}
	templateContext := invoiceTemplateContext(entity, deliveryProfile)
	wording, err := s.resolveInvoiceWording(ctx, &invoiceWordingParams{
		TenantInfo: req.TenantInfo,
		Entity:     entity,
		Profile:    deliveryProfile,
		Context:    templateContext,
	})
	if err != nil {
		return nil, nil, err
	}

	body := wording.Body.Value
	basis := &sendBasis{profile: deliveryProfile, message: wording.Message}
	fromEmail := billingSenderEmail(profile)
	headers := resolveDeliveryHeaders(fromEmail, deliveryProfile.Email)

	plan := &invoiceports.InvoiceSendPlan{
		EDI:                s.edi.PlanFor(ctx, entity, req.TenantInfo),
		InvoiceID:          entity.ID,
		ProviderLimitBytes: providerLimit(profile),
		EstimatedBodyBytes: int64(
			len(wording.Subject.Value)+len(body),
		) + emailports.BodyAllowanceBytes,
		Parts:                make([]*invoiceports.InvoiceSendPlanPart, 0),
		Warnings:             make([]string, 0),
		Errors:               make([]string, 0),
		Recipients:           recipients,
		FromEmail:            fromEmail,
		Headers:              headers,
		OpenTracking:         deliveryProfile.Email != nil && deliveryProfile.Email.ReadReceipt,
		Subject:              wording.Subject.Value,
		Body:                 body,
		InvoicePDFDocumentID: entity.PDFDocumentID,
	}
	plan.Warnings = append(plan.Warnings, templateWarnings("subject", wording.Subject.Unknown)...)
	plan.Warnings = append(plan.Warnings, templateWarnings("body", wording.Body.Unknown)...)
	plan.Errors = append(plan.Errors, sendEligibilityErrors(entity, deliveryProfile.Customer)...)
	replaced, err := s.hiddenAsReplaced(ctx, entity)
	if err != nil {
		return nil, nil, err
	}
	if replaced {
		plan.Errors = append(plan.Errors, replacedInvoiceSendError)
	}
	if err = s.checkSuppressedRecipients(ctx, plan, profile, req.TenantInfo); err != nil {
		return nil, nil, err
	}
	if profile == nil {
		plan.Errors = append(
			plan.Errors,
			"Assign an active Billing email profile before sending invoices",
		)
	}
	if len(recipients.To) == 0 {
		plan.Errors = append(plan.Errors, "No invoice recipients are configured")
	}
	if entity.PDFDocumentID.IsNil() {
		plan.Errors = append(plan.Errors, "Invoice PDF has not been generated")
		return plan, basis, nil
	}

	pdfDoc := entity.PDFDocument
	if pdfDoc == nil {
		pdfDoc, err = s.documentForID(ctx, entity.PDFDocumentID, req.TenantInfo)
		if err != nil {
			plan.Errors = append(plan.Errors, "Invoice PDF document could not be loaded")
			//nolint:nilerr // a plan reports delivery blockers in plan.Errors; a
			// missing PDF is one of them, not a failure to produce the plan
			return plan, basis, nil
		}
	}
	pdfAttachment := planAttachment(pdfDoc, true)
	if deliveryProfile.Email != nil &&
		strings.TrimSpace(deliveryProfile.Email.AttachmentName) != "" {
		attachmentResult := renderInvoiceTemplate(
			deliveryProfile.Email.AttachmentName,
			templateContext,
		)
		pdfAttachment.FileName = invoicePDFAttachmentName(attachmentResult.Value, entity)
		plan.Warnings = append(
			plan.Warnings,
			templateWarnings("attachmentName", attachmentResult.Unknown)...)
	}
	if plan.EstimatedBodyBytes+pdfAttachment.EncodedBytes > plan.ProviderLimitBytes {
		plan.Errors = append(plan.Errors, "Invoice PDF exceeds the email provider message limit")
		return plan, basis, nil
	}

	firstPart := &invoiceports.InvoiceSendPlanPart{
		PartNumber:         1,
		EstimatedSizeBytes: plan.EstimatedBodyBytes + pdfAttachment.EncodedBytes,
		Attachments:        []*invoiceports.InvoiceSendPlanAttachment{pdfAttachment},
	}
	plan.Parts = append(plan.Parts, firstPart)

	attachments, err := s.repo.ListAttachments(ctx, invoiceports.ListInvoiceEmailAttemptsRequest{
		InvoiceID:  entity.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	for _, selected := range attachments {
		if selected == nil || !selected.Selected || selected.Document == nil {
			continue
		}
		attachment := planAttachment(selected.Document, false)
		attachment.PageRotations = selected.PageRotations
		if plan.EstimatedBodyBytes+attachment.EncodedBytes > plan.ProviderLimitBytes {
			link := &invoiceports.InvoiceSendPlanDocumentLink{
				DocumentID: selected.Document.ID,
				FileName:   selected.Document.OriginalName,
				SizeBytes:  selected.Document.FileSize,
				Reason:     "Document exceeds the provider attachment limit and will be sent as a signed download link",
			}
			firstPart.Links = append(firstPart.Links, link)
			firstPart.Warnings = append(firstPart.Warnings, link.Reason+": "+link.FileName)
			plan.Warnings = append(plan.Warnings, link.Reason+": "+link.FileName)
			continue
		}
		current := plan.Parts[len(plan.Parts)-1]
		if current.EstimatedSizeBytes+attachment.EncodedBytes > plan.ProviderLimitBytes {
			current = &invoiceports.InvoiceSendPlanPart{
				PartNumber:         len(plan.Parts) + 1,
				EstimatedSizeBytes: plan.EstimatedBodyBytes,
			}
			plan.Parts = append(plan.Parts, current)
		}
		current.Attachments = append(current.Attachments, attachment)
		current.EstimatedSizeBytes += attachment.EncodedBytes
	}
	return plan, basis, nil
}

const sendInFlightTimeout = 15 * time.Minute

func (s *Service) Send(
	ctx context.Context,
	req *invoiceports.InvoiceSendRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoiceSendResult, error) {
	plan, err := s.PlanSend(ctx, &invoiceports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(plan.Errors) > 0 {
		return nil, errortypes.NewValidationError(
			"sendPlan",
			errortypes.ErrInvalid,
			strings.Join(plan.Errors, "; "),
		)
	}

	entity, err := s.repo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	if err = refuseSendInFlight(entity, now); err != nil {
		return nil, err
	}
	previous := *entity
	applySendSnapshot(entity, plan)
	entity.SentByID = invoiceaction.ActorUserID(actor, req.TenantInfo)
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	if err = s.enqueueInvoiceSendWorkflow(ctx, updated, req, actor, now); err != nil {
		updated.SendStatus = invoice.SendStatusFailed
		updated.LastSendError = err.Error()
		_, _ = s.repo.Update(ctx, updated)
		return nil, err
	}

	s.rec.LogAction(ctx,
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery queued",
	)
	s.rec.PublishInvoiceChange(
		ctx,
		updated,
		actor.AuditActor(),
		invoicerealtime.RealtimeActionSendUpdated,
	)
	return &invoiceports.InvoiceSendResult{Invoice: updated, Plan: plan}, nil
}

func (s *Service) SendFromWorkflow(
	ctx context.Context,
	req *invoiceports.InvoiceSendRequest,
	actor *permissionports.RequestActor,
) (result *invoiceports.InvoiceSendResult, err error) {
	defer func() {
		if err != nil {
			s.markInvoiceSendFailed(ctx, req, actor, err)
		}
	}()

	plan, basis, err := s.planSend(ctx, &invoiceports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(plan.Errors) > 0 {
		return nil, errortypes.NewValidationError(
			"sendPlan",
			errortypes.ErrInvalid,
			strings.Join(plan.Errors, "; "),
		)
	}

	profile, err := s.emailRepo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeBilling)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	startedAt := req.StartedAt
	if startedAt <= 0 {
		startedAt = now
	}
	previous := *entity
	applySendSnapshot(entity, plan)
	if _, err = s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}

	attempts := make([]*invoice.EmailAttempt, 0, len(plan.Parts))
	sendErrors := make([]string, 0)
	for _, part := range plan.Parts {
		links, linkedAttachments, linkErr := s.materializePartLinks(ctx, req, actor, part)
		emailAttachments, attachmentErr := s.emailAttachmentsForPlan(
			ctx,
			req.TenantInfo,
			part.Attachments,
		)
		subject := partSubject(plan.Subject, part.PartNumber, len(plan.Parts))
		partBody := plan.Body

		var message *email.Message
		var sendErr error
		switch {
		case linkErr != nil:
			sendErr = linkErr
		case attachmentErr != nil:
			sendErr = attachmentErr
		default:
			message, partBody, sendErr = s.sendPart(ctx, &partSend{
				req:         req,
				entity:      entity,
				plan:        plan,
				part:        part,
				basis:       basis,
				profile:     profile,
				subject:     subject,
				links:       links,
				attachments: emailAttachments,
				startedAt:   startedAt,
			})
		}

		attempt := &invoice.EmailAttempt{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			InvoiceID:      entity.ID,
			AttemptNumber:  len(attempts) + 1,
			PartNumber:     part.PartNumber,
			TotalParts:     len(plan.Parts),
			Status:         invoice.SendStatusSending,
			Provider:       profile.Provider,
			ToRecipients:   plan.Recipients.To,
			CCRecipients:   plan.Recipients.CC,
			BCCRecipients:  plan.Recipients.BCC,
			Subject:        subject,
			Body:           partBody,
			EstimatedSize:  part.EstimatedSizeBytes,
			Warnings:       attemptWarnings(plan, part),
			CreatedByID:    invoiceaction.ActorUserID(actor, req.TenantInfo),
		}
		if sendErr != nil {
			attempt.Status = invoice.SendStatusFailed
			attempt.Error = sendErr.Error()
			attempt.SentAt = nil
			sendErrors = append(sendErrors, sendErr.Error())
		} else if message != nil {
			attempt.EmailMessageID = message.ID
			attempt.ProviderMessageID = message.ProviderMessageID
		}

		attemptAttachments := attemptAttachmentsForPlan(part.Attachments, linkedAttachments)
		createdAttempt, createErr := s.repo.CreateEmailAttempt(ctx, attempt, attemptAttachments)
		if createErr != nil {
			return nil, createErr
		}
		attempts = append(attempts, createdAttempt)
	}

	entity, err = s.repo.GetByID(
		ctx,
		invoiceports.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	entity.SentByID = invoiceaction.ActorUserID(actor, req.TenantInfo)
	switch {
	case len(sendErrors) == 0:
		entity.SendStatus = invoice.SendStatusSending
	case len(sendErrors) < len(plan.Parts):
		entity.SendStatus = invoice.SendStatusSending
		entity.LastSendError = strings.Join(sliceutils.DedupeStrings(sendErrors), "; ")
	default:
		entity.SendStatus = invoice.SendStatusFailed
		entity.LastSendError = strings.Join(sliceutils.DedupeStrings(sendErrors), "; ")
	}
	entity.LastSendWarning = strings.Join(plan.Warnings, "; ")
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.rec.LogAction(ctx,
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery attempted",
	)
	s.rec.PublishInvoiceChange(
		ctx,
		updated,
		actor.AuditActor(),
		invoicerealtime.RealtimeActionSendUpdated,
	)
	return &invoiceports.InvoiceSendResult{Invoice: updated, Plan: plan, Attempts: attempts}, nil
}

type partSend struct {
	req         *invoiceports.InvoiceSendRequest
	entity      *invoice.Invoice
	plan        *invoiceports.InvoiceSendPlan
	part        *invoiceports.InvoiceSendPlanPart
	basis       *sendBasis
	profile     *email.Profile
	subject     string
	links       []invoicetemplate.InvoiceEmailLink
	attachments []emailports.EmailAttachment
	startedAt   int64
}

func (s *Service) sendPart(ctx context.Context, p *partSend) (*email.Message, string, error) {
	rendered, err := s.renderInvoiceEmail(ctx, &invoiceEmailRenderParams{
		TenantInfo: p.req.TenantInfo,
		Entity:     p.entity,
		Profile:    p.basis.profile,
		PartLabel:  partLabel(p.part.PartNumber, len(p.plan.Parts)),
		Message:    p.basis.message,
		Subject:    p.subject,
		Links:      p.links,
	})
	if err != nil {
		return nil, p.plan.Body, err
	}

	message, err := s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:   p.req.TenantInfo,
		ProfileID:    p.profile.ID,
		Type:         email.TypeInvoice,
		Purpose:      email.PurposeBilling,
		To:           p.plan.Recipients.To,
		CC:           p.plan.Recipients.CC,
		BCC:          p.plan.Recipients.BCC,
		FromEmail:    p.plan.FromEmail,
		Subject:      rendered.Subject,
		HTML:         rendered.HTML,
		Text:         rendered.Text,
		Attachments:  p.attachments,
		Headers:      p.plan.Headers,
		OpenTracking: p.plan.OpenTracking,
		IdempotencyKey: fmt.Sprintf(
			"invoice-%s-part-%d-%d",
			p.entity.ID,
			p.part.PartNumber,
			p.startedAt,
		),
	})
	return message, rendered.Text, err
}

func (s *Service) enqueueInvoiceSendWorkflow(
	ctx context.Context,
	entity *invoice.Invoice,
	req *invoiceports.InvoiceSendRequest,
	actor *permissionports.RequestActor,
	now int64,
) error {
	auditActor := actor.AuditActor()
	workflowID := fmt.Sprintf(
		"invoice-send-%s-%s-%s-%d",
		entity.OrganizationID.String(),
		entity.BusinessUnitID.String(),
		entity.ID.String(),
		now,
	)

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Send invoice email " + entity.Number,
		},
		billingcontract.SendInvoiceEmailWorkflowName,
		&billingcontract.SendInvoiceEmailPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				UserID:         invoiceaction.ActorUserID(actor, req.TenantInfo),
				Timestamp:      now,
			},
			InvoiceID:     entity.ID,
			PrincipalType: auditActor.PrincipalType,
			PrincipalID:   auditActor.PrincipalID,
			APIKeyID:      auditActor.APIKeyID,
		},
	)
	return err
}

func (s *Service) markInvoiceSendFailed(
	ctx context.Context,
	req *invoiceports.InvoiceSendRequest,
	actor *permissionports.RequestActor,
	sendErr error,
) {
	if req == nil || sendErr == nil {
		return
	}
	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		s.l.Error("failed to load invoice after send failure", zap.Error(err))
		return
	}
	previous := *entity
	entity.SendStatus = invoice.SendStatusFailed
	entity.LastSendError = sendErr.Error()
	entity.SentByID = invoiceaction.ActorUserID(actor, req.TenantInfo)
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to mark invoice send failure", zap.Error(err))
		return
	}
	s.rec.LogAction(ctx,
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery failed",
	)
	s.rec.PublishInvoiceChange(
		ctx,
		updated,
		actor.AuditActor(),
		invoicerealtime.RealtimeActionSendUpdated,
	)
}

// applySendSnapshot freezes what was sent onto the invoice.
//
// Template-rendered wording is deliberately not frozen. The snapshot columns are
// also the first tier resolveSubject reads, so storing a render would mean a later
// edit to the organization's template never reached a re-sent invoice — and the
// frozen copy would come back through the ad-hoc {number} engine rather than
// html/template, losing the layout with it. What was actually sent is still
// recoverable from the email message and the send attempts.
func (s *Service) checkSuppressedRecipients(
	ctx context.Context,
	plan *invoiceports.InvoiceSendPlan,
	profile *email.Profile,
	tenantInfo pagination.TenantInfo,
) error {
	if profile == nil || len(plan.Recipients.To) == 0 {
		return nil
	}
	sender, err := s.emailSenders.ResolveSender(ctx, &emailports.SendEmailRequest{
		TenantInfo: tenantInfo,
		ProfileID:  profile.ID,
		Type:       email.TypeInvoice,
		Purpose:    email.PurposeBilling,
		To:         plan.Recipients.To,
		CC:         plan.Recipients.CC,
		BCC:        plan.Recipients.BCC,
	})
	if err != nil {
		return err
	}
	if sender == nil || len(sender.Suppressed) == 0 {
		return nil
	}
	listed := strings.Join(sender.Suppressed, ", ")
	if sender.Refused {
		plan.Errors = append(
			plan.Errors,
			"Every To recipient is on the suppression list after a hard bounce or complaint: "+listed+
				". Remove them from the suppression list or change the recipients.",
		)
		return nil
	}
	plan.Warnings = append(
		plan.Warnings,
		listed+" is on the suppression list after a hard bounce or complaint and will be skipped.",
	)
	return nil
}

func sendEligibilityErrors(entity *invoice.Invoice, cus *customer.Customer) []string {
	errs := make([]string, 0, 2)
	if entity.Status != invoice.StatusPosted {
		errs = append(errs, "Post the invoice before sending it. Draft invoices are never emailed.")
	}
	if cus != nil && cus.BillingProfile != nil && !cus.BillingProfile.EmailInvoiceEnabled {
		errs = append(
			errs,
			"Email delivery is turned off for this customer. Turn on Email invoice in the customer's billing profile to send it.",
		)
	}
	return errs
}

func refuseSendInFlight(entity *invoice.Invoice, now int64) error {
	if entity.SendStatus != invoice.SendStatusSending {
		return nil
	}
	if now-entity.UpdatedAt >= int64(sendInFlightTimeout/time.Second) {
		return nil
	}
	return errortypes.NewBusinessError(
		"Invoice {0} is already being sent. Wait for that send to finish; if it has not finished within {1} minutes, you can send it again.",
		entity.Number,
		int(sendInFlightTimeout/time.Minute),
	)
}

func applySendSnapshot(entity *invoice.Invoice, plan *invoiceports.InvoiceSendPlan) {
	entity.SendStatus = invoice.SendStatusSending
	entity.LastSendError = ""
	entity.LastSendWarning = strings.Join(plan.Warnings, "; ")
}

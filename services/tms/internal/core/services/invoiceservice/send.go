package invoiceservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func (s *Service) PlanSend(
	ctx context.Context,
	req *servicesports.InvoiceSendPlanRequest,
) (*servicesports.InvoiceSendPlan, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseVoidedInvoice(entity, "sent"); err != nil {
		return nil, err
	}
	deliveryProfile, err := s.resolveDeliveryProfile(ctx, resolveDeliveryProfileParams{
		Entity:                      entity,
		TenantInfo:                  req.TenantInfo,
		IncludeShipmentDetails:      true,
		IncludeCustomer:             true,
		IncludeCustomerEmailProfile: true,
	})
	if err != nil {
		return nil, err
	}
	var profile *email.Profile
	if s.emailRepo != nil {
		profile, err = s.emailRepo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeBilling)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
	}
	recipients := resolveRecipients(entity, deliveryProfile.Email)
	if err = s.resolveDeliveryOrganization(ctx, deliveryProfile, req.TenantInfo); err != nil {
		return nil, err
	}
	templateContext := invoiceTemplateContext(entity, deliveryProfile)
	wording, err := s.resolveInvoiceWording(ctx, &invoiceWordingParams{
		TenantInfo: req.TenantInfo,
		Entity:     entity,
		Profile:    deliveryProfile,
		Context:    templateContext,
	})
	if err != nil {
		return nil, err
	}

	body := wording.Body.Value
	if deliveryProfile.Email != nil && deliveryProfile.Email.IncludeShipmentDetail {
		body = appendShipmentDetail(body, entity, deliveryProfile.Shipment)
	}
	fromEmail, fromErr := resolveFromEmail(profile, deliveryProfile.Email)
	headers := resolveDeliveryHeaders(fromEmail, deliveryProfile.Email)
	senderNotice := describeInvoiceSender(profile, deliveryProfile.Customer, fromEmail)

	plan := &servicesports.InvoiceSendPlan{
		EDI:                s.ediPlanFor(ctx, entity, req.TenantInfo),
		InvoiceID:          entity.ID,
		ProviderLimitBytes: providerLimit(profile),
		EstimatedBodyBytes: int64(
			len(wording.Subject.Value)+len(body),
		) + defaultBodyOverheadBytes,
		Parts:                make([]*servicesports.InvoiceSendPlanPart, 0),
		Warnings:             make([]string, 0),
		Errors:               make([]string, 0),
		Recipients:           recipients,
		FromEmail:            fromEmail,
		FromEmailOrigin:      senderNotice.Origin,
		Headers:              headers,
		OpenTracking:         deliveryProfile.Email != nil && deliveryProfile.Email.ReadReceipt,
		Subject:              wording.Subject.Value,
		Body:                 body,
		InvoicePDFDocumentID: entity.PDFDocumentID,
		// Only the template tier produces HTML. A draft or a profile comment is
		// free text, and the send path wraps it the way it always has.
		BodyHTML:     wording.HTML,
		FromTemplate: wording.FromTemplate,
	}
	plan.Warnings = append(plan.Warnings, templateWarnings("subject", wording.Subject.Unknown)...)
	plan.Warnings = append(plan.Warnings, templateWarnings("body", wording.Body.Unknown)...)
	if senderNotice.Warning != "" {
		plan.Warnings = append(plan.Warnings, senderNotice.Warning)
	}
	if fromErr != nil {
		plan.Errors = append(plan.Errors, fromErr.Error())
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
		return plan, nil
	}

	pdfDoc := entity.PDFDocument
	if pdfDoc == nil {
		pdfDoc, err = s.documentForID(ctx, entity.PDFDocumentID, req.TenantInfo)
		if err != nil {
			plan.Errors = append(plan.Errors, "Invoice PDF document could not be loaded")
			//nolint:nilerr // a plan reports delivery blockers in plan.Errors; a
			// missing PDF is one of them, not a failure to produce the plan
			return plan, nil
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
		return plan, nil
	}

	firstPart := &servicesports.InvoiceSendPlanPart{
		PartNumber:         1,
		EstimatedSizeBytes: plan.EstimatedBodyBytes + pdfAttachment.EncodedBytes,
		Attachments:        []*servicesports.InvoiceSendPlanAttachment{pdfAttachment},
	}
	plan.Parts = append(plan.Parts, firstPart)

	attachments, err := s.repo.ListAttachments(ctx, repositories.ListInvoiceEmailAttemptsRequest{
		InvoiceID:  entity.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	for _, selected := range attachments {
		if selected == nil || !selected.Selected || selected.Document == nil {
			continue
		}
		attachment := planAttachment(selected.Document, false)
		if plan.EstimatedBodyBytes+attachment.EncodedBytes > plan.ProviderLimitBytes {
			link := &servicesports.InvoiceSendPlanDocumentLink{
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
			current = &servicesports.InvoiceSendPlanPart{
				PartNumber:         len(plan.Parts) + 1,
				EstimatedSizeBytes: plan.EstimatedBodyBytes,
			}
			plan.Parts = append(plan.Parts, current)
		}
		current.Attachments = append(current.Attachments, attachment)
		current.EstimatedSizeBytes += attachment.EncodedBytes
	}
	return plan, nil
}

const sendInFlightTimeout = 15 * time.Minute

func (s *Service) Send(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceSendResult, error) {
	if s.documentService == nil {
		return nil, errortypes.NewBusinessError("Invoice document delivery is not configured")
	}
	if !s.workflowStarter.Enabled() {
		return nil, servicesports.ErrWorkflowStarterDisabled
	}

	plan, err := s.PlanSend(ctx, &servicesports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
		BaseURL:    req.BaseURL,
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
		repositories.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
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
	entity.SentByID = actorUserID(actor, req.TenantInfo)
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

	s.logAction(
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery queued",
	)
	return &servicesports.InvoiceSendResult{Invoice: updated, Plan: plan}, nil
}

func (s *Service) SendFromWorkflow(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
) (result *servicesports.InvoiceSendResult, err error) {
	defer func() {
		if err != nil {
			s.markInvoiceSendFailed(ctx, req, actor, err)
		}
	}()

	if s.emailService == nil || s.documentService == nil {
		return nil, errortypes.NewBusinessError("Invoice email delivery is not configured")
	}

	plan, err := s.PlanSend(ctx, &servicesports.InvoiceSendPlanRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
		BaseURL:    req.BaseURL,
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

	if s.emailRepo == nil {
		return nil, errortypes.NewBusinessError("Email profile repository is not configured")
	}
	profile, err := s.emailRepo.GetAssignedProfile(ctx, req.TenantInfo, email.PurposeBilling)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(
		ctx,
		repositories.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
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

	deliveryProfile, err := s.splitSendProfile(ctx, req, entity, plan)
	if err != nil {
		return nil, err
	}

	attempts := make([]*invoice.EmailAttempt, 0, len(plan.Parts))
	sendErrors := make([]string, 0)
	for _, part := range plan.Parts {
		partBody, linkedAttachments, linkErr := s.materializePartLinks(ctx, req, actor, part)
		if strings.TrimSpace(partBody) == "" {
			partBody = plan.Body
		} else {
			partBody = strings.TrimSpace(plan.Body) + "\n\n" + partBody
		}
		emailAttachments, attachmentErr := s.emailAttachmentsForPlan(
			ctx,
			req.TenantInfo,
			part.Attachments,
		)
		partHTML := s.partBodyHTML(ctx, &partBodyHTMLParams{
			Request:  req,
			Entity:   entity,
			Profile:  deliveryProfile,
			Plan:     plan,
			Part:     part,
			PartBody: partBody,
		})

		var message *email.Message
		var sendErr error
		if linkErr == nil && attachmentErr == nil {
			message, sendErr = s.emailService.Send(ctx, &servicesports.SendEmailRequest{
				TenantInfo:      req.TenantInfo,
				ProfileID:       profile.ID,
				Purpose:         email.PurposeBilling,
				To:              plan.Recipients.To,
				CC:              plan.Recipients.CC,
				BCC:             plan.Recipients.BCC,
				FromEmail:       plan.FromEmail,
				FromEmailOrigin: plan.FromEmailOrigin,
				Subject:         partSubject(plan.Subject, part.PartNumber, len(plan.Parts)),
				HTML:            partHTML,
				Text:            partBody,
				Attachments:     emailAttachments,
				Headers:         plan.Headers,
				OpenTracking:    plan.OpenTracking,
				IdempotencyKey: fmt.Sprintf(
					"invoice-%s-part-%d-%d",
					entity.ID,
					part.PartNumber,
					startedAt,
				),
			})
		} else if attachmentErr != nil {
			sendErr = attachmentErr
		} else {
			sendErr = linkErr
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
			Subject:        partSubject(plan.Subject, part.PartNumber, len(plan.Parts)),
			Body:           partBody,
			EstimatedSize:  part.EstimatedSizeBytes,
			Warnings:       attemptWarnings(plan, part),
			CreatedByID:    actorUserID(actor, req.TenantInfo),
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
		repositories.GetInvoiceByIDRequest{ID: req.InvoiceID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	entity.SentByID = actorUserID(actor, req.TenantInfo)
	if len(sendErrors) == 0 {
		entity.SendStatus = invoice.SendStatusSending
	} else if len(sendErrors) < len(plan.Parts) {
		entity.SendStatus = invoice.SendStatusSending
		entity.LastSendError = strings.Join(sliceutils.DedupeStrings(sendErrors), "; ")
	} else {
		entity.SendStatus = invoice.SendStatusFailed
		entity.LastSendError = strings.Join(sliceutils.DedupeStrings(sendErrors), "; ")
	}
	entity.LastSendWarning = strings.Join(plan.Warnings, "; ")
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAction(
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery attempted",
	)
	return &servicesports.InvoiceSendResult{Invoice: updated, Plan: plan, Attempts: attempts}, nil
}

func (s *Service) enqueueInvoiceSendWorkflow(
	ctx context.Context,
	entity *invoice.Invoice,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
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
		billingjobs.SendInvoiceEmailWorkflowName,
		&billingjobs.SendInvoiceEmailPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				UserID:         actorUserID(actor, req.TenantInfo),
				Timestamp:      now,
			},
			InvoiceID:     entity.ID,
			BaseURL:       req.BaseURL,
			PrincipalType: auditActor.PrincipalType,
			PrincipalID:   auditActor.PrincipalID,
			APIKeyID:      auditActor.APIKeyID,
		},
	)
	return err
}

func (s *Service) markInvoiceSendFailed(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
	sendErr error,
) {
	if req == nil || sendErr == nil {
		return
	}
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
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
	entity.SentByID = actorUserID(actor, req.TenantInfo)
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to mark invoice send failure", zap.Error(err))
		return
	}
	s.logAction(
		updated,
		actor.AuditActor(),
		permission.OpSubmit,
		&previous,
		updated,
		"Invoice email delivery failed",
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

func applySendSnapshot(entity *invoice.Invoice, plan *servicesports.InvoiceSendPlan) {
	entity.SendStatus = invoice.SendStatusSending
	entity.LastSendError = ""
	entity.LastSendWarning = strings.Join(plan.Warnings, "; ")
	if !plan.FromTemplate {
		entity.EmailSubjectSnapshot = plan.Subject
		entity.EmailBodySnapshot = plan.Body
	}
	entity.EmailToSnapshot = plan.Recipients.To
	entity.EmailCCSnapshot = plan.Recipients.CC
	entity.EmailBCCSnapshot = plan.Recipients.BCC
}

// partBodyHTML picks the HTML for one message of a send.
//
// A template-rendered invoice re-renders per part so the wording can name which
// message this is; anything the part appended — secure download links — is wrapped
// and appended after it, because those links are generated per part and cannot be
// inside the template. A render failure here falls back to wrapping the plain text
// rather than failing a send whose plan already validated: the recipient gets the
// invoice, unstyled, instead of nothing.
// splitSendProfile loads what a split send needs to re-render its wording per
// message, and nothing when the send is a single email.
//
// The profile is read again rather than carried on the plan because the plan
// crosses a workflow boundary as JSON, and one extra read on the rare invoice too
// large for a single email is cheaper than putting a customer record in a Temporal
// payload.
func (s *Service) splitSendProfile(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	entity *invoice.Invoice,
	plan *servicesports.InvoiceSendPlan,
) (*invoiceDeliveryProfile, error) {
	if !plan.FromTemplate || len(plan.Parts) <= 1 {
		return nil, nil //nolint:nilnil // no profile is needed for a single message
	}

	profile, err := s.resolveDeliveryProfile(ctx, resolveDeliveryProfileParams{
		Entity:                      entity,
		TenantInfo:                  req.TenantInfo,
		IncludeShipmentDetails:      true,
		IncludeCustomer:             true,
		IncludeCustomerEmailProfile: true,
	})
	if err != nil {
		return nil, err
	}
	if err = s.resolveDeliveryOrganization(ctx, profile, req.TenantInfo); err != nil {
		return nil, err
	}

	return profile, nil
}

// refuseVoidedInvoice keeps a voided document out of the customer's hands: it
// is still readable inside Trenova, but it is never rendered or delivered.
func refuseVoidedInvoice(entity *invoice.Invoice, action string) error {
	if entity == nil || entity.Status != invoice.StatusVoided {
		return nil
	}

	return errortypes.NewValidationError(
		"invoiceId",
		errortypes.ErrInvalidOperation,
		"Invoice {0} has been voided and cannot be {1}",
		entity.Number,
		action,
	)
}

// ediPlanFor is the EDI half of a send plan. A failure to resolve it must not
// stop the email plan, so it degrades to an unconfigured plan with the error.
func (s *Service) ediPlanFor(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) *servicesports.InvoiceEDISendPlan {
	plans, err := s.ResolveEDISendPlans(ctx, &servicesports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: tenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		s.l.Warn("failed to resolve invoice EDI plan", zap.Error(err))
		return &servicesports.InvoiceEDISendPlan{
			InvoiceID: entity.ID,
			Status:    entity.EDISendStatus,
			Blockers:  []string{"EDI configuration could not be read: " + err.Error()},
		}
	}
	if plan := plans[entity.ID]; plan != nil {
		return plan
	}

	return &servicesports.InvoiceEDISendPlan{InvoiceID: entity.ID, Status: entity.EDISendStatus}
}

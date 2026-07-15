package invoiceservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/mail"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/encodingutils"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	postmarkMessageLimitBytes = int64(10 * 1024 * 1024)
	resendMessageLimitBytes   = int64(40 * 1024 * 1024)
	defaultBodyOverheadBytes  = int64(16 * 1024)
	shareTokenTTL             = 14 * 24 * time.Hour
	invoiceDocumentTypeCode   = "INVOICE"
)

var invoiceTemplateVariablePattern = regexp.MustCompile(
	`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}|\{\s*([A-Za-z0-9_.]+)\s*\}`,
)

type invoiceDeliveryProfile struct {
	Customer       *customer.Customer
	Email          *customer.CustomerEmailProfile
	Organization   *tenant.Organization
	Shipment       *shipment.Shipment
	BillingControl *tenant.BillingControl
}

type resolveDeliveryProfileParams struct {
	Entity                        *invoice.Invoice
	TenantInfo                    pagination.TenantInfo
	IncludeShipmentDetails        bool
	IncludeCustomer               bool
	IncludeCustomerState          bool
	IncludeCustomerBillingProfile bool
	IncludeCustomerEmailProfile   bool
	IncludeBillingControl         bool
}

type invoiceTemplateResult struct {
	Value   string
	Unknown []string
}

func (s *Service) CreateFromShipments(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if len(req.ShipmentIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrRequired,
			"One shipment is required",
		)
	}
	if len(req.ShipmentIDs) > 1 {
		return s.groupedInvoiceFromShipments(ctx, req, actor)
	}

	shp, err := s.shipmentRepo.GetByID(
		ctx,
		expandedShipmentByIDRequest(req.ShipmentIDs[0], req.TenantInfo),
	)
	if err != nil {
		return nil, err
	}
	if shp.Status != shipment.StatusReadyToInvoice && shp.Status != shipment.StatusCompleted {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"Shipment must be completed or ready to invoice",
		)
	}

	exists, err := s.billingQueueRepo.ExistsByShipmentAndType(
		ctx,
		req.TenantInfo,
		shp.ID,
		billingqueue.BillTypeInvoice,
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errortypes.NewConflictError(
			"A billing queue item already exists for this shipment",
		)
	}

	number, err := s.sequenceGenerator.GenerateInvoiceNumber(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	queueItem, err := s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     shp.ID,
		Status:         billingqueue.StatusApproved,
		BillType:       billingqueue.BillTypeInvoice,
		Number:         number,
	})
	if err != nil {
		return nil, err
	}

	result, err := s.CreateFromApprovedBillingQueueItem(
		ctx,
		&servicesports.CreateInvoiceFromBillingQueueRequest{
			BillingQueueItemID: queueItem.ID,
			TenantInfo:         req.TenantInfo,
		},
		actor,
	)
	if err != nil {
		return nil, err
	}

	return result.Invoice, nil
}

// collectBillableLegs gathers the shipments of an order that are ready to invoice,
// loading each leg's full detail (charges) and guarding against double-billing and
// mixed customers.
func (s *Service) collectBillableLegs(
	ctx context.Context,
	ord *order.Order,
	tenantInfo pagination.TenantInfo,
) ([]*shipment.Shipment, error) {
	legs := make([]*shipment.Shipment, 0, len(ord.Shipments))
	for _, leg := range ord.Shipments {
		if leg == nil {
			continue
		}
		if leg.Status != shipment.StatusReadyToInvoice && leg.Status != shipment.StatusCompleted {
			continue
		}
		if leg.CustomerID != ord.CustomerID {
			return nil, errortypes.NewValidationError(
				"orderId",
				errortypes.ErrInvalid,
				"All legs of a grouped invoice must share the order's customer",
			)
		}

		exists, err := s.billingQueueRepo.ExistsByShipmentAndType(
			ctx,
			tenantInfo,
			leg.ID,
			billingqueue.BillTypeInvoice,
		)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errortypes.NewConflictError(
				"A billing queue item already exists for a leg of this order",
			)
		}

		full, err := s.shipmentRepo.GetByID(
			ctx,
			expandedShipmentByIDRequest(leg.ID, tenantInfo),
		)
		if err != nil {
			return nil, err
		}
		legs = append(legs, full)
	}

	return legs, nil
}

// groupedInvoiceFromShipments resolves the single order shared by the given legs and
// delegates to CreateFromOrder. It rejects legs that are not all under one order.
func (s *Service) groupedInvoiceFromShipments(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	var orderID pulid.ID
	for _, shipmentID := range req.ShipmentIDs {
		shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         shipmentID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if shp.OrderID.IsNil() {
			return nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"Grouped invoicing requires every shipment to belong to an order",
			)
		}
		if orderID.IsNil() {
			orderID = shp.OrderID
		} else if orderID != shp.OrderID {
			return nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"All shipments in a grouped invoice must belong to the same order",
			)
		}
	}

	return s.CreateFromOrder(ctx, &servicesports.CreateInvoiceFromOrderRequest{
		OrderID:    orderID,
		TenantInfo: req.TenantInfo,
	}, actor)
}

// CreateFromOrder issues a single grouped invoice covering every billable leg of an
// order. Each billable leg gets its own approved billing-queue item (all carrying the
// order id); the first is the anchor whose id backs the invoice header's single-valued
// FK and idempotency lookup.
func (s *Service) CreateFromOrder(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if req.OrderID.IsNil() {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrRequired,
			"Order is required",
		)
	}

	ord, err := s.orderRepo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              req.OrderID,
		TenantInfo:      req.TenantInfo,
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}

	legs, err := s.collectBillableLegs(ctx, ord, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if len(legs) == 0 {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalid,
			"Order has no legs that are completed or ready to invoice",
		)
	}

	number, err := s.sequenceGenerator.GenerateInvoiceNumber(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	var anchor *billingqueue.BillingQueueItem
	for _, leg := range legs {
		// Only the anchor item carries the invoice number; sibling items leave it null
		// (the billing-queue number is globally unique). The whole group is correlated
		// by OrderID instead.
		itemNumber := ""
		if anchor == nil {
			itemNumber = number
		}

		item, itemErr := s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ShipmentID:     leg.ID,
			OrderID:        ord.ID,
			Status:         billingqueue.StatusApproved,
			BillType:       billingqueue.BillTypeInvoice,
			Number:         itemNumber,
		})
		if itemErr != nil {
			return nil, itemErr
		}
		if anchor == nil {
			anchor = item
		}
	}

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         ord.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if err != nil {
		return nil, err
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	charges, err := s.orderRepo.ListCharges(ctx, req.TenantInfo, ord.ID)
	if err != nil {
		return nil, err
	}

	entity := s.buildInvoiceEntityForOrder(anchor, ord, legs, charges, cus, control)
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Grouped invoice created from order",
	)
	s.publishInvalidation(ctx, created, auditActor, "created", created)

	return created, nil
}

func (s *Service) UpdateDraft(
	ctx context.Context,
	req *servicesports.UpdateInvoiceDraftRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
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
	if entity.Status != invoice.StatusDraft {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only draft invoices can be updated",
		)
	}

	previous := *entity
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

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	if req.AttachmentIDs != nil {
		if _, err = s.repo.UpsertAttachments(ctx, repositories.UpsertInvoiceAttachmentsRequest{
			InvoiceID:      updated.ID,
			DocumentIDs:    *req.AttachmentIDs,
			OrganizationID: updated.OrganizationID,
			BusinessUnitID: updated.BusinessUnitID,
			TenantInfo:     req.TenantInfo,
		}); err != nil {
			return nil, err
		}
	}

	s.logAction(
		updated,
		actor.AuditActor(),
		permission.OpUpdate,
		&previous,
		updated,
		"Invoice draft delivery metadata updated",
	)
	return s.repo.GetByID(
		ctx,
		repositories.GetInvoiceByIDRequest{ID: updated.ID, TenantInfo: req.TenantInfo},
	)
}

func (s *Service) RenderPreview(
	ctx context.Context,
	req *servicesports.InvoicePreviewRequest,
) (*servicesports.InvoicePreviewResult, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	deliveryProfile, err := s.resolveDeliveryProfile(ctx, resolveDeliveryProfileParams{
		Entity:                 entity,
		TenantInfo:             req.TenantInfo,
		IncludeShipmentDetails: true,
		IncludeCustomer:        true,
		IncludeCustomerState:   true,
		IncludeBillingControl:  true,
	})
	if err != nil {
		return nil, err
	}
	if err = s.resolveDeliveryOrganization(ctx, deliveryProfile, req.TenantInfo); err != nil {
		return nil, err
	}
	return invoicePreviewForEntity(ctx, entity, deliveryProfile, s.storage)
}

func (s *Service) GeneratePDF(
	ctx context.Context,
	req *servicesports.InvoicePreviewRequest,
	actor *servicesports.RequestActor,
) (*servicesports.GenerateInvoicePDFResult, error) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Invoice PDF generation requires workflow processing to be enabled",
		).WithInternal(servicesports.ErrWorkflowStarterDisabled)
	}
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	userID := actorUserID(actor, req.TenantInfo)
	if userID.IsNil() {
		return nil, errortypes.NewValidationError(
			"userId",
			errortypes.ErrRequired,
			"User ID is required to generate invoice PDFs",
		)
	}

	workflowID := fmt.Sprintf(
		"invoice-pdf-generate-%s-%s",
		req.InvoiceID.String(),
		pulid.MustNew("wf_").String(),
	)
	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: temporaltype.TaskQueueBilling.String(),
			StaticSummary: fmt.Sprintf(
				"Generating invoice PDF %s",
				req.InvoiceID.String(),
			),
		},
		billingjobs.GenerateInvoicePDFWorkflowName,
		&billingjobs.GenerateInvoicePDFPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				UserID:         userID,
				Timestamp:      timeutils.NowUnix(),
			},
			InvoiceID:     req.InvoiceID,
			BaseURL:       req.BaseURL,
			PrincipalType: actor.PrincipalType,
			PrincipalID:   actor.PrincipalID,
			APIKeyID:      actor.APIKeyID,
		},
	)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to start invoice PDF generation").
			WithInternal(err)
	}

	return &servicesports.GenerateInvoicePDFResult{
		InvoiceID:     req.InvoiceID,
		WorkflowID:    run.GetID(),
		WorkflowRunID: run.GetRunID(),
		Status:        "Queued",
	}, nil
}

func (s *Service) AutoSendInvoiceAfterPDFGeneration(
	ctx context.Context,
	req *servicesports.AutoSendInvoiceAfterPDFGenerationRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceSendResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
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
	if cus.BillingProfile == nil || !cus.BillingProfile.AutoSendInvoiceOnGeneration {
		return nil, nil
	}

	switch entity.SendStatus {
	case invoice.SendStatusNotSent, invoice.SendStatusFailed:
		return s.SendFromWorkflow(ctx, &servicesports.InvoiceSendRequest{
			InvoiceID:  req.InvoiceID,
			TenantInfo: req.TenantInfo,
			BaseURL:    req.BaseURL,
		}, actor)
	case invoice.SendStatusSending, invoice.SendStatusSent, invoice.SendStatusPartiallySent:
		return nil, nil
	default:
		return nil, nil
	}
}

func invoicePreviewForEntity(
	ctx context.Context,
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
	storageClient storage.Client,
) (*servicesports.InvoicePreviewResult, error) {
	content, err := renderInvoicePDF(ctx, entity, deliveryProfile, storageClient)
	if err != nil {
		return nil, err
	}

	return &servicesports.InvoicePreviewResult{
		Content:     content,
		ContentType: "application/pdf",
		FileName:    invoicePDFName(entity),
		SizeBytes:   int64(len(content)),
	}, nil
}

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
	subjectResult := resolveSubject(entity, deliveryProfile.Email, templateContext)
	bodyResult := resolveBody(entity, deliveryProfile.Email, templateContext)
	body := bodyResult.Value
	if deliveryProfile.Email != nil && deliveryProfile.Email.IncludeShipmentDetail {
		body = appendShipmentDetail(body, entity, deliveryProfile.Shipment)
	}
	fromEmail, fromErr := resolveFromEmail(profile, deliveryProfile.Email)
	headers := resolveDeliveryHeaders(fromEmail, deliveryProfile.Email)

	plan := &servicesports.InvoiceSendPlan{
		InvoiceID:            entity.ID,
		ProviderLimitBytes:   providerLimit(profile),
		EstimatedBodyBytes:   int64(len(subjectResult.Value)+len(body)) + defaultBodyOverheadBytes,
		Parts:                make([]*servicesports.InvoiceSendPlanPart, 0),
		Warnings:             make([]string, 0),
		Errors:               make([]string, 0),
		Recipients:           recipients,
		FromEmail:            fromEmail,
		Headers:              headers,
		OpenTracking:         deliveryProfile.Email != nil && deliveryProfile.Email.ReadReceipt,
		Subject:              subjectResult.Value,
		Body:                 body,
		InvoicePDFDocumentID: entity.PDFDocumentID,
	}
	plan.Warnings = append(plan.Warnings, templateWarnings("subject", subjectResult.Unknown)...)
	plan.Warnings = append(plan.Warnings, templateWarnings("body", bodyResult.Unknown)...)
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
	previous := *entity
	applySendSnapshot(entity, plan)
	if _, err = s.repo.Update(ctx, entity); err != nil {
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
		var message *email.Message
		var sendErr error
		if linkErr == nil && attachmentErr == nil {
			message, sendErr = s.emailService.Send(ctx, &servicesports.SendEmailRequest{
				TenantInfo:   req.TenantInfo,
				ProfileID:    profile.ID,
				Purpose:      email.PurposeBilling,
				To:           plan.Recipients.To,
				CC:           plan.Recipients.CC,
				BCC:          plan.Recipients.BCC,
				FromEmail:    plan.FromEmail,
				Subject:      partSubject(plan.Subject, part.PartNumber, len(plan.Parts)),
				HTML:         bodyHTML(partBody),
				Text:         partBody,
				Attachments:  emailAttachments,
				Headers:      plan.Headers,
				OpenTracking: plan.OpenTracking,
				IdempotencyKey: fmt.Sprintf(
					"invoice-%s-part-%d-%d",
					entity.ID,
					part.PartNumber,
					now,
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
		entity.LastSendError = strings.Join(sendErrors, "; ")
	} else {
		entity.SendStatus = invoice.SendStatusFailed
		entity.LastSendError = strings.Join(sendErrors, "; ")
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

func (s *Service) ListEmailAttempts(
	ctx context.Context,
	req repositories.ListInvoiceEmailAttemptsRequest,
) (*pagination.ListResult[*invoice.EmailAttempt], error) {
	return s.repo.ListEmailAttempts(ctx, req)
}

func (s *Service) DownloadSharedDocument(
	ctx context.Context,
	req *servicesports.DownloadInvoiceDocumentRequest,
) (*servicesports.DownloadInvoiceDocumentResult, error) {
	if s.documentService == nil {
		return nil, errortypes.NewBusinessError("Document service is not configured")
	}
	tokenHash := tokenHash(req.Token)
	share, err := s.repo.GetDocumentShareToken(
		ctx,
		repositories.GetInvoiceDocumentShareTokenRequest{
			TokenHash: tokenHash,
		},
	)
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	if share.RevokedAt != nil || share.ExpiresAt <= now {
		return nil, errortypes.NewNotFoundError("Document link is no longer available")
	}
	content, err := s.documentService.GetDownloadContent(ctx, repositories.GetDocumentByIDRequest{
		ID: share.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: share.OrganizationID,
			BuID:  share.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	body, err := readDocumentBody(content.Body, "shared document")
	if err != nil {
		return nil, err
	}

	share.DownloadedAt = &now
	if _, err = s.repo.UpdateDocumentShareToken(ctx, share); err != nil {
		return nil, err
	}

	return &servicesports.DownloadInvoiceDocumentResult{
		FileName:      content.Document.OriginalName,
		ContentType:   content.ContentType,
		ContentLength: int64(len(body)),
		ContentDisposition: fileutils.ContentDisposition(
			"attachment",
			content.Document.OriginalName,
		),
		Body: body,
	}, nil
}

func (s *Service) resolveDeliveryProfile(
	ctx context.Context,
	params resolveDeliveryProfileParams,
) (*invoiceDeliveryProfile, error) {
	result := &invoiceDeliveryProfile{}
	entity := params.Entity
	if entity == nil {
		return result, nil
	}

	if entity.Customer != nil {
		result.Customer = entity.Customer
		if params.IncludeCustomerEmailProfile {
			result.Email = entity.Customer.EmailProfile
		}
		result.Organization = entity.Customer.Organization
	}
	if entity.Shipment != nil {
		result.Shipment = entity.Shipment
	}
	if params.IncludeCustomer && s.customerRepo != nil && entity.CustomerID.IsNotNil() {
		cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         entity.CustomerID,
			TenantInfo: params.TenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeState:          params.IncludeCustomerState,
				IncludeBillingProfile: params.IncludeCustomerBillingProfile,
				IncludeEmailProfile:   params.IncludeCustomerEmailProfile,
			},
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
		if cus != nil {
			result.Customer = cus
			if params.IncludeCustomerEmailProfile {
				result.Email = cus.EmailProfile
			}
			if cus.Organization != nil {
				result.Organization = cus.Organization
			}
		}
	}
	if params.IncludeShipmentDetails && s.shipmentRepo != nil && entity.ShipmentID.IsNotNil() {
		shp, err := s.shipmentRepo.GetByID(
			ctx,
			expandedShipmentByIDRequest(entity.ShipmentID, params.TenantInfo),
		)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
		if shp != nil {
			result.Shipment = shp
		}
	}
	if params.IncludeBillingControl && s.billingRepo != nil {
		control, err := s.billingRepo.GetByOrgID(ctx, params.TenantInfo.OrgID)
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
		if control != nil {
			result.BillingControl = control
		}
	}
	return result, nil
}

func basicShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *repositories.GetShipmentByIDRequest {
	return shipmentByIDRequest(id, tenantInfo, false)
}

func expandedShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *repositories.GetShipmentByIDRequest {
	return shipmentByIDRequest(id, tenantInfo, true)
}

func shipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	expandShipmentDetails bool,
) *repositories.GetShipmentByIDRequest {
	return &repositories.GetShipmentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: expandShipmentDetails,
		},
	}
}

func (s *Service) resolveDeliveryOrganization(
	ctx context.Context,
	deliveryProfile *invoiceDeliveryProfile,
	tenantInfo pagination.TenantInfo,
) error {
	if deliveryProfile == nil || deliveryProfile.Organization != nil || s.organizationRepo == nil {
		return nil
	}
	org, err := s.organizationRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if org != nil {
		deliveryProfile.Organization = org
	}
	return nil
}

func resolveRecipients(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
) servicesports.InvoiceSendRecipients {
	recipients := servicesports.InvoiceSendRecipients{
		To:  normalizeRecipients(entity.EmailToSnapshot),
		CC:  normalizeRecipients(entity.EmailCCSnapshot),
		BCC: normalizeRecipients(entity.EmailBCCSnapshot),
	}
	if len(recipients.To) > 0 {
		return recipients
	}
	if emailProfile == nil {
		return recipients
	}
	recipients.To = splitRecipients(emailProfile.ToRecipients)
	recipients.CC = splitRecipients(emailProfile.CCRecipients)
	recipients.BCC = splitRecipients(emailProfile.BCCRecipients)
	return recipients
}

func resolveSubject(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	context map[string]string,
) invoiceTemplateResult {
	if strings.TrimSpace(entity.EmailSubjectSnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailSubjectSnapshot, context)
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Subject) != "" {
		return renderInvoiceTemplate(emailProfile.Subject, context)
	}
	return invoiceTemplateResult{Value: "Invoice " + entity.Number}
}

func resolveBody(
	entity *invoice.Invoice,
	emailProfile *customer.CustomerEmailProfile,
	context map[string]string,
) invoiceTemplateResult {
	if strings.TrimSpace(entity.EmailBodySnapshot) != "" {
		return renderInvoiceTemplate(entity.EmailBodySnapshot, context)
	}
	if emailProfile != nil && strings.TrimSpace(emailProfile.Comment) != "" {
		return renderInvoiceTemplate(emailProfile.Comment, context)
	}
	var b strings.Builder
	b.WriteString("Please find invoice ")
	b.WriteString(entity.Number)
	b.WriteString(" attached.")
	if entity.Memo != "" {
		b.WriteString("\n\n")
		b.WriteString(entity.Memo)
	}
	return invoiceTemplateResult{Value: b.String()}
}

func invoiceTemplateContext(
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
) map[string]string {
	values := map[string]string{
		"number":              entity.Number,
		"invoice.number":      entity.Number,
		"invoiceNumber":       entity.Number,
		"invoice.date":        unixDate(entity.InvoiceDate),
		"invoiceDate":         unixDate(entity.InvoiceDate),
		"invoice.dueDate":     unixDatePtr(entity.DueDate),
		"dueDate":             unixDatePtr(entity.DueDate),
		"invoice.paymentTerm": string(entity.PaymentTerm),
		"paymentTerm":         string(entity.PaymentTerm),
		"invoice.total": moneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		"invoiceTotal": moneyString(
			entity.CurrencyCode,
			entity.TotalAmount.StringFixed(2),
		),
		"invoice.currency":        entity.CurrencyCode,
		"currency":                entity.CurrencyCode,
		"customer":                entity.BillToName,
		"customer.name":           entity.BillToName,
		"customerName":            entity.BillToName,
		"customer.code":           entity.BillToCode,
		"customerCode":            entity.BillToCode,
		"company":                 "",
		"organization.name":       "",
		"organizationName":        "",
		"shipment.pro":            entity.ShipmentProNumber,
		"shipmentPro":             entity.ShipmentProNumber,
		"shipment.bol":            entity.ShipmentBOL,
		"shipmentBol":             entity.ShipmentBOL,
		"shipment.serviceDate":    unixDatePtr(entity.ServiceDate),
		"serviceDate":             unixDatePtr(entity.ServiceDate),
		"remittance.instructions": entity.RemittanceInstructions,
		"remittanceInstructions":  entity.RemittanceInstructions,
	}
	if deliveryProfile == nil {
		return values
	}
	if deliveryProfile.Organization != nil {
		values["company"] = deliveryProfile.Organization.Name
		values["organization.name"] = deliveryProfile.Organization.Name
		values["organizationName"] = deliveryProfile.Organization.Name
	}
	if deliveryProfile.Customer != nil {
		values["customer.name"] = stringutils.FirstNonEmpty(
			deliveryProfile.Customer.Name,
			values["customer.name"],
		)
		values["customer.code"] = stringutils.FirstNonEmpty(
			deliveryProfile.Customer.Code,
			values["customer.code"],
		)
		values["customer"] = values["customer.name"]
		values["customerName"] = values["customer.name"]
		values["customerCode"] = values["customer.code"]
		if deliveryProfile.Organization == nil && deliveryProfile.Customer.Organization != nil {
			values["company"] = deliveryProfile.Customer.Organization.Name
			values["organization.name"] = deliveryProfile.Customer.Organization.Name
			values["organizationName"] = deliveryProfile.Customer.Organization.Name
		}
	}
	if deliveryProfile.Shipment != nil {
		shp := deliveryProfile.Shipment
		values["shipment.pro"] = stringutils.FirstNonEmpty(shp.ProNumber, values["shipment.pro"])
		values["shipment.bol"] = stringutils.FirstNonEmpty(shp.BOL, values["shipment.bol"])
		values["shipment.pickupDate"] = unixDatePtr(shp.ActualShipDate)
		values["shipment.deliveryDate"] = unixDatePtr(shp.ActualDeliveryDate)
		values["shipment.origin"] = shipmentOrigin(shp)
		values["shipment.destination"] = shipmentDestination(shp)
		values["shipmentPro"] = values["shipment.pro"]
		values["shipmentBol"] = values["shipment.bol"]
		values["pickupDate"] = values["shipment.pickupDate"]
		values["deliveryDate"] = values["shipment.deliveryDate"]
		values["origin"] = values["shipment.origin"]
		values["destination"] = values["shipment.destination"]
	}
	return values
}

func renderInvoiceTemplate(template string, values map[string]string) invoiceTemplateResult {
	unknown := make([]string, 0)
	rendered := invoiceTemplateVariablePattern.ReplaceAllStringFunc(
		template,
		func(match string) string {
			parts := invoiceTemplateVariablePattern.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			key := parts[1]
			if key == "" {
				key = parts[2]
			}
			value, ok := values[key]
			if !ok {
				unknown = append(unknown, key)
				return match
			}
			return value
		},
	)
	return invoiceTemplateResult{
		Value:   strings.TrimSpace(rendered),
		Unknown: uniqueStrings(unknown),
	}
}

func templateWarnings(field string, unknown []string) []string {
	if len(unknown) == 0 {
		return nil
	}
	warnings := make([]string, 0, len(unknown))
	for _, variable := range unknown {
		warnings = append(warnings, "Unknown "+field+" template variable: "+variable)
	}
	return warnings
}

func resolveFromEmail(
	profile *email.Profile,
	emailProfile *customer.CustomerEmailProfile,
) (string, error) {
	if profile == nil {
		return "", nil
	}
	fromEmail := strings.TrimSpace(profile.SenderEmail)
	if emailProfile == nil || strings.TrimSpace(emailProfile.FromEmail) == "" {
		return fromEmail, nil
	}
	override := strings.TrimSpace(emailProfile.FromEmail)
	parsed, err := mail.ParseAddress(override)
	if err != nil || parsed.Address != override {
		return fromEmail, errortypes.NewValidationError(
			"fromEmail",
			errortypes.ErrInvalid,
			"Customer invoice sender email is invalid",
		)
	}
	return override, nil
}

func resolveDeliveryHeaders(
	fromEmail string,
	emailProfile *customer.CustomerEmailProfile,
) map[string]string {
	if emailProfile == nil || !emailProfile.ReadReceipt || strings.TrimSpace(fromEmail) == "" {
		return nil
	}
	return map[string]string{"Disposition-Notification-To": strings.TrimSpace(fromEmail)}
}

func invoicePDFAttachmentName(rendered string, entity *invoice.Invoice) string {
	name := fileutils.SafeFilename(rendered)
	if strings.TrimSpace(name) == "" || name == "." {
		name = invoicePDFName(entity)
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = strings.TrimSpace(strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '-'
		default:
			return r
		}
	}, base))
	if base == "" {
		base = strings.TrimSuffix(invoicePDFName(entity), ".pdf")
	}
	return base + ".pdf"
}

func appendShipmentDetail(body string, entity *invoice.Invoice, shp *shipment.Shipment) string {
	detail := shipmentDetailBlock(entity, shp)
	if detail == "" {
		return body
	}
	if strings.TrimSpace(body) == "" {
		return detail
	}
	return strings.TrimSpace(body) + "\n\n" + detail
}

func shipmentDetailBlock(entity *invoice.Invoice, shp *shipment.Shipment) string {
	lines := []string{"Shipment Detail"}
	appendDetailLine := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, label+": "+strings.TrimSpace(value))
		}
	}
	appendDetailLine("PRO", stringutils.FirstNonEmpty(entity.ShipmentProNumber, shipmentPro(shp)))
	appendDetailLine("BOL", stringutils.FirstNonEmpty(entity.ShipmentBOL, shipmentBOL(shp)))
	appendDetailLine("Route", shipmentRoute(shp))
	appendDetailLine("Service Date", unixDatePtr(entity.ServiceDate))
	if shp != nil {
		appendDetailLine("Pickup", unixDatePtr(shp.ActualShipDate))
		appendDetailLine("Delivery", unixDatePtr(shp.ActualDeliveryDate))
		appendDetailLine("Commodities", commoditySummary(shp))
		appendDetailLine("Pieces", int64PtrString(shp.Pieces))
		appendDetailLine("Weight", int64PtrString(shp.Weight))
	}
	if len(entity.Lines) > 0 {
		lines = append(lines, "Charges:")
		for _, line := range entity.Lines {
			if line == nil {
				continue
			}
			lines = append(lines, fmt.Sprintf(
				"- %s: %s",
				line.Description,
				moneyString(entity.CurrencyCode, line.Amount.StringFixed(2)),
			))
		}
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func shipmentRoute(shp *shipment.Shipment) string {
	origin := shipmentOrigin(shp)
	destination := shipmentDestination(shp)
	if origin == "" && destination == "" {
		return ""
	}
	return origin + " -> " + destination
}

func shipmentOrigin(shp *shipment.Shipment) string {
	return stopLocationName(firstPickupStop(shp))
}

func shipmentDestination(shp *shipment.Shipment) string {
	return stopLocationName(finalDeliveryStop(shp))
}

func firstPickupStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsOriginStop() },
		preferLowerStopSequence,
	)
}

func firstDeliveryStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsDestinationStop() },
		preferLowerStopSequence,
	)
}

func finalDeliveryStop(shp *shipment.Shipment) *shipment.Stop {
	return selectShipmentStop(
		shp,
		func(stop *shipment.Stop) bool { return stop.IsDestinationStop() },
		preferHigherStopSequence,
	)
}

func selectShipmentStop(
	shp *shipment.Shipment,
	matches func(*shipment.Stop) bool,
	prefer func(candidate, selected *shipment.Stop) bool,
) *shipment.Stop {
	if shp == nil {
		return nil
	}
	var selected *shipment.Stop
	for _, move := range shp.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if !matches(stop) {
				continue
			}
			if prefer(stop, selected) {
				selected = stop
			}
		}
	}
	return selected
}

func preferLowerStopSequence(candidate, selected *shipment.Stop) bool {
	return selected == nil || candidate.Sequence < selected.Sequence
}

func preferHigherStopSequence(candidate, selected *shipment.Stop) bool {
	return selected == nil || candidate.Sequence > selected.Sequence
}

func stopLocationName(stop *shipment.Stop) string {
	if stop == nil {
		return ""
	}
	if stop.Location != nil {
		return strings.TrimSpace(strings.Join([]string{
			stop.Location.Name,
			stop.Location.City,
			stop.Location.PostalCode,
		}, " "))
	}
	return strings.TrimSpace(stop.AddressLine)
}

func commoditySummary(shp *shipment.Shipment) string {
	if shp == nil || len(shp.Commodities) == 0 {
		return ""
	}
	parts := make([]string, 0, len(shp.Commodities))
	for _, item := range shp.Commodities {
		if item == nil {
			continue
		}
		name := "Commodity"
		if item.Commodity != nil && strings.TrimSpace(item.Commodity.Name) != "" {
			name = item.Commodity.Name
		}
		parts = append(parts, fmt.Sprintf("%s (%d pcs, %d lbs)", name, item.Pieces, item.Weight))
	}
	return strings.Join(parts, "; ")
}

func shipmentPro(shp *shipment.Shipment) string {
	if shp == nil {
		return ""
	}
	return shp.ProNumber
}

func shipmentBOL(shp *shipment.Shipment) string {
	if shp == nil {
		return ""
	}
	return shp.BOL
}

func int64PtrString(value *int64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func uniqueStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, item := range input {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func (s *Service) documentForID(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*document.Document, error) {
	if s.documentService == nil {
		return nil, errortypes.NewBusinessError("Document service is not configured")
	}
	return s.documentService.Get(
		ctx,
		repositories.GetDocumentByIDRequest{ID: documentID, TenantInfo: tenantInfo},
	)
}

func (s *Service) emailAttachmentsForPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planned []*servicesports.InvoiceSendPlanAttachment,
) ([]servicesports.EmailAttachment, error) {
	result := make([]servicesports.EmailAttachment, 0, len(planned))
	for _, item := range planned {
		content, err := s.documentService.GetDownloadContent(
			ctx,
			repositories.GetDocumentByIDRequest{
				ID:         item.DocumentID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("prepare attachment %s: %w", item.FileName, err)
		}
		body, err := io.ReadAll(content.Body)
		closeErr := content.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read attachment %s: %w", item.FileName, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close attachment %s: %w", item.FileName, closeErr)
		}
		result = append(result, servicesports.EmailAttachment{
			FileName:    item.FileName,
			ContentType: item.ContentType,
			Content:     body,
			SizeBytes:   int64(len(body)),
		})
	}
	return result, nil
}

func readDocumentBody(body io.ReadCloser, label string) ([]byte, error) {
	content, err := io.ReadAll(body)
	closeErr := body.Close()
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to read " + label).WithInternal(err)
	}
	if closeErr != nil {
		return nil, errortypes.NewDatabaseError("Failed to close " + label).WithInternal(closeErr)
	}
	return content, nil
}

func (s *Service) materializePartLinks(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
	part *servicesports.InvoiceSendPlanPart,
) (string, []*invoice.EmailAttemptAttachment, error) {
	body := strings.TrimSpace(partBodyBase(part))
	linked := make([]*invoice.EmailAttemptAttachment, 0, len(part.Links))
	for _, link := range part.Links {
		rawToken, hash, err := newShareToken()
		if err != nil {
			return body, linked, err
		}
		share, err := s.repo.CreateDocumentShareToken(ctx, &invoice.DocumentShareToken{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			InvoiceID:      req.InvoiceID,
			DocumentID:     link.DocumentID,
			TokenHash:      hash,
			ExpiresAt:      time.Now().Add(shareTokenTTL).Unix(),
			CreatedByID:    actorUserID(actor, req.TenantInfo),
		})
		if err != nil {
			return body, linked, err
		}
		link.URL = signedDocumentURL(req.BaseURL, rawToken)
		body += "\n\nSupporting document link: " + link.FileName + "\n" + link.URL
		linked = append(linked, &invoice.EmailAttemptAttachment{
			DocumentID:   link.DocumentID,
			FileName:     link.FileName,
			SizeBytes:    link.SizeBytes,
			EncodedBytes: encodingutils.EncodedBase64Size(link.SizeBytes),
			Method:       invoice.AttachmentDeliveryMethodLink,
			ShareTokenID: share.ID,
			Reason:       link.Reason,
		})
	}
	return body, linked, nil
}

func attemptAttachmentsForPlan(
	planned []*servicesports.InvoiceSendPlanAttachment,
	linked []*invoice.EmailAttemptAttachment,
) []*invoice.EmailAttemptAttachment {
	result := make([]*invoice.EmailAttemptAttachment, 0, len(planned)+len(linked))
	for _, item := range planned {
		result = append(result, &invoice.EmailAttemptAttachment{
			DocumentID:   item.DocumentID,
			FileName:     item.FileName,
			ContentType:  item.ContentType,
			SizeBytes:    item.SizeBytes,
			EncodedBytes: item.EncodedBytes,
			Method:       invoice.AttachmentDeliveryMethodAttached,
		})
	}
	result = append(result, linked...)
	return result
}

func attemptWarnings(
	plan *servicesports.InvoiceSendPlan,
	part *servicesports.InvoiceSendPlanPart,
) []string {
	capacity := len(plan.Warnings) + len(part.Warnings)
	if plan.OpenTracking {
		capacity++
	}
	if len(plan.Headers) > 0 {
		capacity++
	}
	result := make([]string, 0, capacity)
	result = append(result, plan.Warnings...)
	result = append(result, part.Warnings...)
	if plan.OpenTracking {
		result = append(result, "Read receipt/open tracking requested")
	}
	if len(plan.Headers) > 0 {
		result = append(result, "Custom email headers requested")
	}
	return uniqueStrings(result)
}

func planAttachment(
	doc *document.Document,
	invoicePDF bool,
) *servicesports.InvoiceSendPlanAttachment {
	return &servicesports.InvoiceSendPlanAttachment{
		DocumentID:   doc.ID,
		FileName:     doc.OriginalName,
		ContentType:  doc.FileType,
		SizeBytes:    doc.FileSize,
		EncodedBytes: encodingutils.EncodedBase64Size(doc.FileSize),
		InvoicePDF:   invoicePDF,
	}
}

func providerLimit(profile *email.Profile) int64 {
	if profile != nil && profile.Provider == email.ProviderResend {
		return resendMessageLimitBytes
	}
	return postmarkMessageLimitBytes
}

func splitRecipients(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	return normalizeRecipients(parts)
}

func normalizeRecipients(input []string) []string {
	normalized := stringutils.NormalizeEmailAddresses(input)
	result := make([]string, 0, len(normalized))
	seen := make(map[string]struct{}, len(input))
	for _, item := range normalized {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func applySendSnapshot(entity *invoice.Invoice, plan *servicesports.InvoiceSendPlan) {
	entity.SendStatus = invoice.SendStatusSending
	entity.LastSendError = ""
	entity.LastSendWarning = strings.Join(plan.Warnings, "; ")
	entity.EmailSubjectSnapshot = plan.Subject
	entity.EmailBodySnapshot = plan.Body
	entity.EmailToSnapshot = plan.Recipients.To
	entity.EmailCCSnapshot = plan.Recipients.CC
	entity.EmailBCCSnapshot = plan.Recipients.BCC
}

func invoicePDFName(entity *invoice.Invoice) string {
	number := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(entity.Number)
	return "invoice-" + number + ".pdf"
}

func unixDate(value int64) string {
	if value == 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format("2006-01-02")
}

func unixDatePtr(value *int64) string {
	if value == nil {
		return ""
	}
	return unixDate(*value)
}

func moneyString(currency, amount string) string {
	if currency == "" {
		currency = "USD"
	}
	return currency + " " + amount
}

func actorForDocument(
	actor *servicesports.RequestActor,
	tenantInfo pagination.TenantInfo,
) servicesports.RequestActor {
	if actor == nil {
		return servicesports.RequestActor{
			UserID:         tenantInfo.UserID,
			PrincipalID:    tenantInfo.UserID,
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
		}
	}
	return *actor
}

func actorUserID(actor *servicesports.RequestActor, tenantInfo pagination.TenantInfo) pulid.ID {
	if actor != nil && actor.UserID.IsNotNil() {
		return actor.UserID
	}
	return tenantInfo.UserID
}

func partSubject(subject string, part, total int) string {
	if total <= 1 {
		return subject
	}
	return fmt.Sprintf("%s (%d of %d)", subject, part, total)
}

func partBodyBase(part *servicesports.InvoiceSendPlanPart) string {
	if len(part.Links) == 0 {
		return ""
	}
	return "Some supporting documents are available through secure download links below."
}

func bodyHTML(body string) string {
	escaped := html.EscapeString(body)
	return "<p>" + strings.ReplaceAll(escaped, "\n", "<br>") + "</p>"
}

func newShareToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	return raw, tokenHash(raw), nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func signedDocumentURL(baseURL, token string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return "/api/v1/billing/invoices/shared-documents/" + url.PathEscape(token) + "/download/"
	}
	return baseURL + "/api/v1/billing/invoices/shared-documents/" + url.PathEscape(
		token,
	) + "/download/"
}

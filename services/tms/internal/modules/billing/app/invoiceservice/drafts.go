package invoiceservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
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
	if err = refuseNonDraftUpdate(entity); err != nil {
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
		invoiceports.GetInvoiceByIDRequest{ID: updated.ID, TenantInfo: req.TenantInfo},
	)
}

func (s *Service) RenderPreview(
	ctx context.Context,
	req *invoiceports.InvoicePreviewRequest,
) (*invoiceports.InvoicePreviewResult, error) {
	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseVoidedInvoice(entity, "rendered"); err != nil {
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
	return s.invoicePreviewForEntity(ctx, entity, deliveryProfile, req.TenantInfo)
}

func (s *Service) GeneratePDF(
	ctx context.Context,
	req *invoiceports.InvoicePreviewRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.GenerateInvoicePDFResult, error) {
	_, userID, err := s.planPDFGeneration(ctx, req, actor)
	if err != nil {
		return nil, err
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
		billingcontract.GenerateInvoicePDFWorkflowName,
		&billingcontract.GenerateInvoicePDFPayload{
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

	return &invoiceports.GenerateInvoicePDFResult{
		InvoiceID:     req.InvoiceID,
		WorkflowID:    run.GetID(),
		WorkflowRunID: run.GetRunID(),
		Status:        "Queued",
	}, nil
}

func (s *Service) AutoSendInvoiceAfterPDFGeneration(
	ctx context.Context,
	req *invoiceports.AutoSendInvoiceAfterPDFGenerationRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.InvoiceSendResult, error) {
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

	entity, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = refuseVoidedInvoice(entity, "sent"); err != nil {
		return nil, err
	}

	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         entity.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	if !autoSendsOnGeneration(cus, entity) {
		return nil, nil
	}

	return s.SendFromWorkflow(ctx, &invoiceports.InvoiceSendRequest{
		InvoiceID:  req.InvoiceID,
		TenantInfo: req.TenantInfo,
		BaseURL:    req.BaseURL,
	}, actor)
}

// invoicePreviewForEntity prints an invoice through the organization's template.
//
// Which template that is — a customer assignment, the organization default, or
// the built-in — is the resolver's decision, so a customer with a bespoke
// invoice layout gets it here and on the send path without either caller
// knowing the difference.
func (s *Service) invoicePreviewForEntity(
	ctx context.Context,
	entity *invoice.Invoice,
	deliveryProfile *invoiceDeliveryProfile,
	tenantInfo pagination.TenantInfo,
) (*invoiceports.InvoicePreviewResult, error) {
	if s.templates == nil {
		return nil, errortypes.NewBusinessError(
			"Invoice rendering is not configured on this deployment",
		)
	}

	data, err := s.contextBuilder.BuildFrom(ctx, entity, deliveryProfile)
	if err != nil {
		return nil, err
	}

	var customerID *pulid.ID
	if entity.CustomerID.IsNotNil() {
		id := entity.CustomerID
		customerID = &id
	}

	rendered, err := s.templates.RenderDocument(ctx, &documenttemplateports.RenderDocumentRequest{
		TenantInfo:  tenantInfo,
		Kind:        documenttemplate.KindInvoicePDF,
		CustomerID:  customerID,
		Data:        data,
		ReferenceID: entity.ID,
		Title:       "Invoice " + entity.Number,
	})
	if err != nil {
		return nil, err
	}

	return &invoiceports.InvoicePreviewResult{
		Content:     rendered.PDF,
		ContentType: "application/pdf",
		FileName:    invoicePDFName(entity),
		SizeBytes:   int64(len(rendered.PDF)),
		Rendered:    rendered,
	}, nil
}

package billingjobs

import (
	"bytes"
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	InvoiceService   services.InvoiceService
	InvoiceRepo      repositories.InvoiceRepository
	DocumentService  services.InvoiceDocumentService
	UploadService    services.DocumentUploadService
	DocumentTypeRepo repositories.DocumentTypeRepository
	AuditService     services.AuditService
	Logger           *zap.Logger
}

type Activities struct {
	invoiceService   services.InvoiceService
	invoiceRepo      repositories.InvoiceRepository
	documentService  services.InvoiceDocumentService
	uploadService    services.DocumentUploadService
	documentTypeRepo repositories.DocumentTypeRepository
	auditService     services.AuditService
	logger           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		invoiceService:   p.InvoiceService,
		invoiceRepo:      p.InvoiceRepo,
		documentService:  p.DocumentService,
		uploadService:    p.UploadService,
		documentTypeRepo: p.DocumentTypeRepo,
		auditService:     p.AuditService,
		logger:           p.Logger.Named("billing-activities"),
	}
}

func (a *Activities) AutoPostInvoiceActivity(
	ctx context.Context,
	payload *AutoPostInvoicePayload,
) (*AutoPostInvoiceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	current, err := a.invoiceService.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if current.Status == invoice.StatusPosted {
		return &AutoPostInvoiceResult{
			InvoiceID:     current.ID,
			PostedAt:      derefInt64(current.PostedAt),
			CompletedAt:   timeutils.NowUnix(),
			AlreadyPosted: true,
		}, nil
	}

	posted, err := a.invoiceService.Post(ctx, &services.PostInvoiceRequest{
		InvoiceID:   payload.InvoiceID,
		TenantInfo:  tenantInfo,
		TriggeredBy: "auto-post-workflow",
	}, &services.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	})
	if err != nil {
		return nil, err
	}

	return &AutoPostInvoiceResult{
		InvoiceID:   posted.ID,
		PostedAt:    derefInt64(posted.PostedAt),
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func (a *Activities) SendInvoiceEmailActivity(
	ctx context.Context,
	payload *SendInvoiceEmailPayload,
) (*SendInvoiceEmailResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	result, err := a.invoiceService.SendFromWorkflow(ctx, &services.InvoiceSendRequest{
		InvoiceID:  payload.InvoiceID,
		TenantInfo: tenantInfo,
		BaseURL:    payload.BaseURL,
	}, &services.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	})
	if err != nil {
		return nil, err
	}

	return &SendInvoiceEmailResult{
		InvoiceID:   result.Invoice.ID,
		SendStatus:  string(result.Invoice.SendStatus),
		Attempts:    len(result.Attempts),
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func (a *Activities) PrepareInvoicePDFUploadActivity(
	ctx context.Context,
	payload *GenerateInvoicePDFPayload,
) (*PrepareInvoicePDFUploadResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	current, err := a.invoiceService.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if current.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required to store generated invoice PDFs as shipment documents",
		)
	}

	preview, err := a.invoiceService.RenderPreview(ctx, &services.InvoicePreviewRequest{
		InvoiceID:  payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	docType, err := a.documentTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
		Code:       "INVOICE",
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	lineageID, err := a.invoicePDFLineageID(ctx, current, tenantInfo)
	if err != nil {
		return nil, err
	}
	actor := invoicePDFActor(payload)
	session, err := a.uploadService.CreateSession(ctx, &services.CreateSessionRequest{
		TenantInfo:     tenantInfo,
		Actor:          actor,
		ResourceID:     current.ShipmentID.String(),
		ResourceType:   "shipment",
		FileName:       preview.FileName,
		FileSize:       preview.SizeBytes,
		ContentType:    preview.ContentType,
		Description:    "Generated customer invoice PDF",
		Tags:           []string{"invoice", "generated"},
		DocumentTypeID: docType.ID.String(),
		LineageID:      lineageID,
	})
	if err != nil {
		return nil, err
	}

	if _, err = a.uploadService.UploadPart(ctx, &services.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(preview.Content),
		Size:       preview.SizeBytes,
	}); err != nil {
		return nil, err
	}

	return &PrepareInvoicePDFUploadResult{
		InvoiceID: payload.InvoiceID,
		SessionID: session.ID,
	}, nil
}

func (a *Activities) CompleteInvoicePDFGenerationActivity(
	ctx context.Context,
	payload *GenerateInvoicePDFPayload,
	documentID pulid.ID,
) (*GenerateInvoicePDFResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	current, err := a.invoiceService.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *current
	current.PDFDocumentID = documentID
	updated, err := a.invoiceRepo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	_ = a.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceInvoice,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         payload.UserID,
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		APIKeyID:       payload.APIKeyID,
		PreviousState:  jsonutils.MustToJSON(&previous),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Invoice PDF generated"))

	if _, err = a.invoiceService.AutoSendInvoiceAfterPDFGeneration(
		ctx,
		&services.AutoSendInvoiceAfterPDFGenerationRequest{
			InvoiceID:  payload.InvoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    payload.BaseURL,
		},
		&services.RequestActor{
			PrincipalType:  payload.PrincipalType,
			PrincipalID:    payload.PrincipalID,
			UserID:         payload.UserID,
			APIKeyID:       payload.APIKeyID,
			BusinessUnitID: payload.BusinessUnitID,
			OrganizationID: payload.OrganizationID,
		},
	); err != nil {
		a.logger.Error(
			"failed to auto-send invoice after PDF generation",
			zap.String("invoiceID", payload.InvoiceID.String()),
			zap.Error(err),
		)
	}

	return &GenerateInvoicePDFResult{
		InvoiceID:   payload.InvoiceID,
		DocumentID:  documentID,
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}

func (a *Activities) invoicePDFLineageID(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	if entity.PDFDocumentID.IsNil() {
		return "", nil
	}

	currentPDF, err := a.documentService.Get(ctx, repositories.GetDocumentByIDRequest{
		ID:         entity.PDFDocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	if currentPDF.ResourceType != "shipment" ||
		currentPDF.ResourceID != entity.ShipmentID.String() ||
		!currentPDF.IsCurrentVersion {
		return "", nil
	}

	return currentPDF.LineageID.String(), nil
}

func invoicePDFActor(payload *GenerateInvoicePDFPayload) services.RequestActor {
	return services.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	}
}

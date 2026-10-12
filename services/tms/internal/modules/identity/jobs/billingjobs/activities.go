package billingjobs

import (
	"bytes"
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	InvoiceService    invoiceports.InvoiceService
	Templates         documenttemplateports.DocumentTemplateResolver
	InvoiceRepo       invoiceports.InvoiceRepository
	DocumentService   documentports.InvoiceDocumentService
	UploadService     documentuploadports.DocumentUploadService
	DocumentTypeRepo  documenttypeports.DocumentTypeRepository
	InvoiceRunSweeper invoicerunports.InvoiceRunSweeper
	Logger            *zap.Logger

	LateChargeService  latechargeports.LateChargeService    `optional:"true"`
	LateChargeRepo     latechargeports.LateChargeRepository `optional:"true"`
	BillingControlRepo tenantports.BillingControlRepository
	EDIService         ediports.EDIService `optional:"true"`
}

type Activities struct {
	invoiceService    invoiceports.InvoiceService
	templates         documenttemplateports.DocumentTemplateResolver
	invoiceRepo       invoiceports.InvoiceRepository
	documentService   documentports.InvoiceDocumentService
	uploadService     documentuploadports.DocumentUploadService
	documentTypeRepo  documenttypeports.DocumentTypeRepository
	invoiceRunSweeper invoicerunports.InvoiceRunSweeper
	logger            *zap.Logger

	lateChargeService  latechargeports.LateChargeService
	lateChargeRepo     latechargeports.LateChargeRepository
	billingControlRepo tenantports.BillingControlRepository
	ediService         ediports.EDIService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		invoiceService:    p.InvoiceService,
		templates:         p.Templates,
		invoiceRepo:       p.InvoiceRepo,
		documentService:   p.DocumentService,
		uploadService:     p.UploadService,
		documentTypeRepo:  p.DocumentTypeRepo,
		invoiceRunSweeper: p.InvoiceRunSweeper,

		lateChargeService:  p.LateChargeService,
		lateChargeRepo:     p.LateChargeRepo,
		billingControlRepo: p.BillingControlRepo,
		ediService:         p.EDIService,
		logger:             p.Logger.Named("billing-activities"),
	}
}

func (a *Activities) AutoPostInvoiceActivity(
	ctx context.Context,
	payload *billingcontract.AutoPostInvoicePayload,
) (*AutoPostInvoiceResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	current, err := a.invoiceService.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, temporaltype.ToDomainRefusal(err)
	}

	if current.Status == invoice.StatusPosted || current.Status == invoice.StatusVoided {
		return &AutoPostInvoiceResult{
			InvoiceID:     current.ID,
			PostedAt:      typeutils.ValueOrZero(current.PostedAt),
			CompletedAt:   timeutils.NowUnix(),
			AlreadyPosted: true,
			Voided:        current.Status == invoice.StatusVoided,
		}, nil
	}

	posted, err := a.invoiceService.Post(ctx, &invoiceports.PostInvoiceRequest{
		InvoiceID:   payload.InvoiceID,
		TenantInfo:  tenantInfo,
		TriggeredBy: "auto-post-workflow",
	}, &permissionports.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	})
	if err != nil {
		return nil, temporaltype.ToDomainRefusal(err)
	}

	return &AutoPostInvoiceResult{
		InvoiceID:   posted.ID,
		PostedAt:    typeutils.ValueOrZero(posted.PostedAt),
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func (a *Activities) SendInvoiceEmailActivity(
	ctx context.Context,
	payload *billingcontract.SendInvoiceEmailPayload,
) (*SendInvoiceEmailResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	stopHeartbeat := temporaltype.HeartbeatEvery(ctx, invoiceSendHeartbeatInterval)
	defer stopHeartbeat()

	result, err := a.invoiceService.SendFromWorkflow(ctx, &invoiceports.InvoiceSendRequest{
		InvoiceID:  payload.InvoiceID,
		TenantInfo: tenantInfo,
		BaseURL:    payload.BaseURL,
		StartedAt:  payload.Timestamp,
	}, &permissionports.RequestActor{
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
	payload *billingcontract.GenerateInvoicePDFPayload,
) (*PrepareInvoicePDFUploadResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	current, err := a.invoiceService.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	resource, err := invoicePDFResource(current)
	if err != nil {
		return nil, err
	}

	preview, err := a.invoiceService.RenderPreview(ctx, &invoiceports.InvoicePreviewRequest{
		InvoiceID:  payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, classifyInvoiceRenderError(err)
	}
	docType, err := a.documentTypeRepo.GetByCode(
		ctx,
		documenttypeports.GetDocumentTypeByCodeRequest{
			Code:       "INVOICE",
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	lineageID, err := a.invoicePDFLineageID(ctx, current, tenantInfo)
	if err != nil {
		return nil, err
	}
	actor := invoicePDFActor(payload)
	session, err := a.uploadService.CreateSession(ctx, &documentuploadports.CreateSessionRequest{
		TenantInfo:     tenantInfo,
		Actor:          actor,
		ResourceID:     resource.ID,
		ResourceType:   resource.Type,
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

	if _, err = a.uploadService.UploadPart(ctx, &documentuploadports.UploadPartRequest{
		TenantInfo: tenantInfo,
		SessionID:  session.ID,
		PartNumber: 1,
		Body:       bytes.NewReader(preview.Content),
		Size:       preview.SizeBytes,
	}); err != nil {
		return nil, err
	}

	a.recordGeneratedInvoicePDF(ctx, &recordInvoicePDFParams{
		TenantInfo: tenantInfo,
		Invoice:    current,
		Preview:    preview,
		UserID:     payload.UserID,
	})

	return &PrepareInvoicePDFUploadResult{
		InvoiceID: payload.InvoiceID,
		SessionID: session.ID,
	}, nil
}

func (a *Activities) CompleteInvoicePDFGenerationActivity(
	ctx context.Context,
	payload *billingcontract.GenerateInvoicePDFPayload,
	documentID pulid.ID,
) (*GenerateInvoicePDFResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	if _, err := a.invoiceService.AttachGeneratedPDF(ctx, &invoiceports.AttachGeneratedPDFRequest{
		InvoiceID:  payload.InvoiceID,
		DocumentID: documentID,
		TenantInfo: tenantInfo,
		Actor: kernelports.AuditActor{
			PrincipalType: payload.PrincipalType,
			PrincipalID:   payload.PrincipalID,
			UserID:        payload.UserID,
			APIKeyID:      payload.APIKeyID,
		},
	}); err != nil {
		return nil, err
	}

	stopHeartbeat := temporaltype.HeartbeatEvery(ctx, invoiceSendHeartbeatInterval)
	defer stopHeartbeat()
	if _, err := a.invoiceService.AutoSendInvoiceAfterPDFGeneration(
		ctx,
		&invoiceports.AutoSendInvoiceAfterPDFGenerationRequest{
			InvoiceID:  payload.InvoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    payload.BaseURL,
			StartedAt:  payload.Timestamp,
		},
		&permissionports.RequestActor{
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

func (a *Activities) invoicePDFLineageID(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	if entity.PDFDocumentID.IsNil() {
		return "", nil
	}

	currentPDF, err := a.documentService.Get(ctx, documentports.GetDocumentByIDRequest{
		ID:         entity.PDFDocumentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	resource, err := invoicePDFResource(entity)
	if err != nil {
		return "", nil //nolint:nilerr // an unanchorable invoice simply starts a new lineage
	}
	if currentPDF.ResourceType != resource.Type ||
		currentPDF.ResourceID != resource.ID ||
		!currentPDF.IsCurrentVersion {
		return "", nil
	}

	return currentPDF.LineageID.String(), nil
}

func invoicePDFActor(
	payload *billingcontract.GenerateInvoicePDFPayload,
) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		BusinessUnitID: payload.BusinessUnitID,
		OrganizationID: payload.OrganizationID,
	}
}

// classifyInvoiceRenderError tells a transient renderer outage apart from a
// template that will never render.
//
// Without this both look the same to Temporal and a broken template burns five
// attempts before surfacing, while a sidecar restart might not be retried at
// all.
func classifyInvoiceRenderError(err error) error {
	if err == nil {
		return nil
	}

	// The sidecar is down or unconfigured. That is exactly what retries are for.
	if errors.Is(err, documenttemplateports.ErrPDFRendererUnavailable) {
		return temporaltype.NewRetryableError(
			"the PDF renderer was unavailable while generating an invoice", err,
		)
	}

	// Field-level errors from the template engine mean the template does not
	// compile or does not render against real data. An administrator has to
	// change it; no number of attempts will.
	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) {
		return temporaltype.NewTemplateInvalidError(
			"the invoice template could not be rendered: "+multiErr.Error(), err,
		)
	}

	return err
}

type recordInvoicePDFParams struct {
	TenantInfo pagination.TenantInfo
	Invoice    *invoice.Invoice
	Preview    *invoiceports.InvoicePreviewResult
	UserID     pulid.ID
}

// recordGeneratedInvoicePDF logs which template version produced the bytes just
// uploaded.
//
// This is the record that answers "what exactly did we send that customer in
// March" once a template has been edited — which is when an invoice dispute asks.
// It is deliberately best-effort: the customer's invoice has already been
// rendered and stored, and failing the workflow over a bookkeeping row would
// turn a missing audit entry into a missing invoice.
func (a *Activities) recordGeneratedInvoicePDF(
	ctx context.Context,
	p *recordInvoicePDFParams,
) {
	if a.templates == nil || p.Preview == nil || p.Preview.Rendered == nil {
		return
	}

	if _, err := a.templates.RecordGeneratedDocument(
		ctx,
		&documenttemplateports.RecordGeneratedDocumentRequest{
			TenantInfo:    p.TenantInfo,
			Kind:          documenttemplate.KindInvoicePDF,
			Rendered:      p.Preview.Rendered,
			ReferenceType: "invoice",
			ReferenceID:   p.Invoice.ID,
			FileName:      p.Preview.FileName,
			// The upload session owns the final object key; this records the
			// logical path so the row is self-describing before finalization.
			FilePath: "invoice/" + p.Invoice.ID.String() + "/" + p.Preview.FileName,
			FileSize: p.Preview.SizeBytes,
			UserID:   p.UserID,
		},
	); err != nil {
		activity.GetLogger(ctx).Warn(
			"could not record the generated invoice PDF", "error", err,
		)
	}
}

// invoicePDFDocumentResource is the record a generated invoice PDF is filed
// under: the shipment for a shipment invoice, the order for an order invoice,
// and the billed customer for statements and memos, which have no single
// shipment to hang a document on.
type invoicePDFDocumentResource struct {
	Type string
	ID   string
}

func invoicePDFResource(entity *invoice.Invoice) (invoicePDFDocumentResource, error) {
	if entity == nil {
		return invoicePDFDocumentResource{}, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrRequired,
			"Invoice is required to store generated invoice PDFs",
		)
	}
	if entity.Status == invoice.StatusVoided {
		return invoicePDFDocumentResource{}, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Voided invoices do not generate documents",
		)
	}
	switch {
	case entity.ShipmentID.IsNotNil():
		return invoicePDFDocumentResource{Type: "shipment", ID: entity.ShipmentID.String()}, nil
	case entity.OrderID.IsNotNil():
		return invoicePDFDocumentResource{Type: "order", ID: entity.OrderID.String()}, nil
	case entity.CustomerID.IsNotNil():
		return invoicePDFDocumentResource{Type: "customer", ID: entity.CustomerID.String()}, nil
	default:
		return invoicePDFDocumentResource{}, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrRequired,
			"Invoice has no shipment, order or customer to store generated PDFs against",
		)
	}
}

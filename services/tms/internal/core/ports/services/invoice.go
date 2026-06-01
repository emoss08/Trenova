package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateInvoiceFromBillingQueueRequest struct {
	BillingQueueItemID pulid.ID
	TenantInfo         pagination.TenantInfo
}

type CreateInvoiceFromBillingQueueResult struct {
	Invoice  *invoice.Invoice
	AutoPost bool
}

type PostInvoiceRequest struct {
	InvoiceID   pulid.ID
	TenantInfo  pagination.TenantInfo
	TriggeredBy string
}

type CreateInvoiceFromShipmentsRequest struct {
	ShipmentIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
}

type UpdateInvoiceDraftRequest struct {
	InvoiceID              pulid.ID
	TenantInfo             pagination.TenantInfo
	Memo                   *string
	RemittanceInstructions *string
	EmailSubject           *string
	EmailBody              *string
	EmailTo                *[]string
	EmailCC                *[]string
	EmailBCC               *[]string
	AttachmentIDs          *[]pulid.ID
}

type InvoicePreviewRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	BaseURL    string
}

type InvoicePreviewResult struct {
	Content     []byte `json:"-"`
	ContentType string `json:"contentType"`
	FileName    string `json:"fileName"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type GenerateInvoicePDFResult struct {
	InvoiceID     pulid.ID `json:"invoiceId"`
	WorkflowID    string   `json:"workflowId"`
	WorkflowRunID string   `json:"workflowRunId"`
	Status        string   `json:"status"`
}

type InvoiceSendPlanRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	BaseURL    string
}

type InvoiceSendRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	BaseURL    string
}

type AutoSendInvoiceAfterPDFGenerationRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	BaseURL    string
}

type InvoiceSendPlan struct {
	InvoiceID            pulid.ID               `json:"invoiceId"`
	ProviderLimitBytes   int64                  `json:"providerLimitBytes"`
	EstimatedBodyBytes   int64                  `json:"estimatedBodyBytes"`
	Parts                []*InvoiceSendPlanPart `json:"parts"`
	Warnings             []string               `json:"warnings"`
	Errors               []string               `json:"errors"`
	Recipients           InvoiceSendRecipients  `json:"recipients"`
	FromEmail            string                 `json:"fromEmail"`
	Headers              map[string]string      `json:"headers"`
	OpenTracking         bool                   `json:"openTracking"`
	Subject              string                 `json:"subject"`
	Body                 string                 `json:"body"`
	InvoicePDFDocumentID pulid.ID               `json:"invoicePdfDocumentId"`
}

type InvoiceSendRecipients struct {
	To  []string `json:"to"`
	CC  []string `json:"cc"`
	BCC []string `json:"bcc"`
}

type InvoiceSendPlanPart struct {
	PartNumber         int                            `json:"partNumber"`
	EstimatedSizeBytes int64                          `json:"estimatedSizeBytes"`
	Attachments        []*InvoiceSendPlanAttachment   `json:"attachments"`
	Links              []*InvoiceSendPlanDocumentLink `json:"links"`
	Warnings           []string                       `json:"warnings"`
}

type InvoiceSendPlanAttachment struct {
	DocumentID   pulid.ID `json:"documentId"`
	FileName     string   `json:"fileName"`
	ContentType  string   `json:"contentType"`
	SizeBytes    int64    `json:"sizeBytes"`
	EncodedBytes int64    `json:"encodedBytes"`
	InvoicePDF   bool     `json:"invoicePdf"`
}

type InvoiceSendPlanDocumentLink struct {
	DocumentID pulid.ID `json:"documentId"`
	FileName   string   `json:"fileName"`
	SizeBytes  int64    `json:"sizeBytes"`
	Reason     string   `json:"reason"`
	URL        string   `json:"url,omitempty"`
}

type InvoiceSendResult struct {
	Invoice  *invoice.Invoice        `json:"invoice"`
	Plan     *InvoiceSendPlan        `json:"plan"`
	Attempts []*invoice.EmailAttempt `json:"attempts"`
}

type DownloadInvoiceDocumentRequest struct {
	Token string `json:"-"`
}

type DownloadInvoiceDocumentResult struct {
	FileName           string `json:"fileName"`
	ContentType        string `json:"contentType"`
	ContentLength      int64  `json:"contentLength"`
	ContentDisposition string `json:"contentDisposition"`
	Body               []byte `json:"-"`
}

type InvoiceService interface {
	List(
		ctx context.Context,
		req *repositories.ListInvoicesRequest,
	) (*pagination.ListResult[*invoice.Invoice], error)
	GetByID(
		ctx context.Context,
		req repositories.GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
	CreateFromApprovedBillingQueueItem(
		ctx context.Context,
		req *CreateInvoiceFromBillingQueueRequest,
		actor *RequestActor,
	) (*CreateInvoiceFromBillingQueueResult, error)
	CreateFromShipments(
		ctx context.Context,
		req *CreateInvoiceFromShipmentsRequest,
		actor *RequestActor,
	) (*invoice.Invoice, error)
	UpdateDraft(
		ctx context.Context,
		req *UpdateInvoiceDraftRequest,
		actor *RequestActor,
	) (*invoice.Invoice, error)
	RenderPreview(
		ctx context.Context,
		req *InvoicePreviewRequest,
	) (*InvoicePreviewResult, error)
	GeneratePDF(
		ctx context.Context,
		req *InvoicePreviewRequest,
		actor *RequestActor,
	) (*GenerateInvoicePDFResult, error)
	AutoSendInvoiceAfterPDFGeneration(
		ctx context.Context,
		req *AutoSendInvoiceAfterPDFGenerationRequest,
		actor *RequestActor,
	) (*InvoiceSendResult, error)
	PlanSend(
		ctx context.Context,
		req *InvoiceSendPlanRequest,
	) (*InvoiceSendPlan, error)
	Send(
		ctx context.Context,
		req *InvoiceSendRequest,
		actor *RequestActor,
	) (*InvoiceSendResult, error)
	SendFromWorkflow(
		ctx context.Context,
		req *InvoiceSendRequest,
		actor *RequestActor,
	) (*InvoiceSendResult, error)
	ListEmailAttempts(
		ctx context.Context,
		req repositories.ListInvoiceEmailAttemptsRequest,
	) (*pagination.ListResult[*invoice.EmailAttempt], error)
	DownloadSharedDocument(
		ctx context.Context,
		req *DownloadInvoiceDocumentRequest,
	) (*DownloadInvoiceDocumentResult, error)
	Post(
		ctx context.Context,
		req *PostInvoiceRequest,
		actor *RequestActor,
	) (*invoice.Invoice, error)
	EnqueueAutoPost(
		ctx context.Context,
		entity *invoice.Invoice,
		actor *RequestActor,
	) error
}

package invoiceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CreateInvoiceFromBillingQueueRequest struct {
	BillingQueueItemID pulid.ID
	TenantInfo         pagination.TenantInfo
	// OffCycleReason acknowledges billing this item on its own for a customer whose
	// freight was supposed to accumulate onto a periodic statement. See
	// CreateInvoiceFromShipmentsRequest.
	OffCycleReason string
	// DeferToStatement asks for a statement customer's freight to be approved onto
	// their statement instead of invoiced.
	//
	// Set by the billing queue's approve action, where approval means "this
	// freight is verified" rather than "invoice it now". Without it, approving a
	// statement customer's shipment would either cut an invoice their schedule
	// says is premature, or be refused outright — and being refused would leave
	// the item unapproved, which is the one state that keeps it off the very
	// statement it belongs to.
	DeferToStatement bool
}

type CreateInvoiceFromBillingQueueResult struct {
	Invoice  *invoice.Invoice
	AutoPost bool
	// DeferredToStatement reports that the item was approved onto its customer's
	// statement rather than invoiced, and Invoice is nil. It is the ordinary
	// outcome of approving a statement customer's freight, not a failure.
	DeferredToStatement bool
}

type PostInvoiceRequest struct {
	InvoiceID   pulid.ID
	TenantInfo  pagination.TenantInfo
	TriggeredBy string
	Automatic   bool
}

// VoidInvoiceRequest takes an invoice out of circulation. A draft is voided in
// place; a posted invoice goes through a full-reversal adjustment so the ledger
// reverses before the invoice is marked. Disposition decides whether the freight
// behind it goes back to the queue or is canceled.
type VoidInvoiceRequest struct {
	InvoiceID   pulid.ID
	TenantInfo  pagination.TenantInfo
	Reason      string
	Disposition invoice.VoidDisposition
}

type AttachGeneratedPDFRequest struct {
	InvoiceID  pulid.ID
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	Actor      kernelports.AuditActor
}

type ReportAutoPostFailureRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	Reason     string
	Actor      kernelports.AuditActor
}

type DiscardDraftForReviewRequest struct {
	BillingQueueItemID pulid.ID
	TenantInfo         pagination.TenantInfo
}

type VoidInvoiceResult struct {
	Invoice *invoice.Invoice `json:"invoice"`
	// AdjustmentID is the full reversal a posted void went through.
	AdjustmentID pulid.ID `json:"adjustmentId"`
	// PendingApproval means the reversal is waiting on an approver; the invoice
	// voids the moment it executes.
	PendingApproval      bool       `json:"pendingApproval"`
	ReleasedQueueItemIDs []pulid.ID `json:"releasedQueueItemIds"`
}

type CreateMemoLineInput struct {
	Description         string          `json:"description"`
	Amount              decimal.Decimal `json:"amount"`
	Quantity            decimal.Decimal `json:"quantity"`
	AccessorialChargeID pulid.ID        `json:"accessorialChargeId"`
}

// CreateMemoRequest raises a credit or debit memo against a customer with no
// shipment behind it: a goodwill credit, a returned-check fee, a late charge.
type CreateMemoRequest struct {
	ID                 pulid.ID
	TenantInfo         pagination.TenantInfo
	CustomerID         pulid.ID
	BillType           billingqueue.BillType
	Lines              []*CreateMemoLineInput
	ReferenceInvoiceID pulid.ID
	Reason             string
	InvoiceDate        int64
	Memo               string
	MemoKind           invoice.MemoKind
	AutoPost           bool
	AutoPostByPolicy   bool `json:"-"`
	// CurrencyCode bills the memo in a currency other than the customer's
	// billing currency, as a late charge on an invoice in that currency must
	// be. Blank means the customer's billing currency.
	CurrencyCode string
}

type CreateInvoiceFromShipmentsRequest struct {
	ShipmentIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
	// OffCycleReason acknowledges that this customer's freight was supposed to
	// accumulate onto a periodic statement. Required only for those customers, and
	// only because taking freight off a statement without a record is how a
	// customer ends up with an invoice they were not expecting and nobody can
	// explain. Ignored for everyone else.
	OffCycleReason string
}

type PlanCreateInvoicesRequest struct {
	OrderID        pulid.ID
	ShipmentIDs    []pulid.ID
	TenantInfo     pagination.TenantInfo
	OffCycleReason string
}

type CreateInvoicesPlan struct {
	Drafts       []*invoice.Invoice
	PrimaryIndex int
}

type InvoiceDraftUpdatePreview struct {
	Before            *invoice.Invoice
	After             *invoice.Invoice
	AttachmentsBefore []pulid.ID
	AttachmentsAfter  []pulid.ID
}

type InvoicePDFGenerationPlan struct {
	Invoice   *invoice.Invoice
	AutoSends bool
}

type InvoiceEDISendPreview struct {
	Invoice *invoice.Invoice
	Plan    *InvoiceEDISendPlan
}

type InvoiceVoidPreview struct {
	Before   *invoice.Invoice
	After    *invoice.Invoice
	Posted   bool
	Reversal *invoiceadjustmentports.InvoiceAdjustmentPreview
}

type CreateInvoiceFromOrderRequest struct {
	OrderID pulid.ID
	// ShipmentIDs optionally restricts the grouped invoice to an explicit subset of the
	// order's billable legs. Empty means every billable leg.
	ShipmentIDs []pulid.ID
	TenantInfo  pagination.TenantInfo
	// OffCycleReason acknowledges billing this order on its own for a customer whose
	// freight was supposed to accumulate onto a periodic statement. See
	// CreateInvoiceFromShipmentsRequest.
	OffCycleReason string
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

	// Rendered carries the template provenance behind these bytes so a caller
	// that stores them can record which version produced them. Nil when the
	// bytes did not come from the template renderer.
	Rendered *documenttemplateports.RenderedDocument `json:"-"`
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
	StartedAt  int64
}

type AutoSendInvoiceAfterPDFGenerationRequest struct {
	InvoiceID  pulid.ID
	TenantInfo pagination.TenantInfo
	BaseURL    string
	StartedAt  int64
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

	// EDI is where the invoice's outbound 210 stands and what blocks it.
	EDI *InvoiceEDISendPlan `json:"edi"`

	// BodyHTML is the organization's template output. It is empty when the
	// subject and body came from an operator's draft or the customer's email
	// profile, because those are free text and the send path wraps them itself.
	BodyHTML string `json:"bodyHtml"`

	// FromTemplate says the wording is the organization's template rather than
	// free text, which decides whether the send path uses BodyHTML.
	FromTemplate bool `json:"fromTemplate"`
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

// CreateInvoicesResult is every invoice a billing action produced. A split-billed
// shipment yields one per payer; Primary is the shipment's own payer's invoice,
// which is what single-invoice callers are shown.
type CreateInvoicesResult struct {
	Invoices []*invoice.Invoice `json:"invoices"`
	Primary  *invoice.Invoice   `json:"primary"`
}

// InvoiceQueryService reads invoices and their email attempts.
type InvoiceQueryService interface {
	List(
		ctx context.Context,
		req *ListInvoicesRequest,
	) (*pagination.ListResult[*invoice.Invoice], error)
	ListConnection(
		ctx context.Context,
		req *ListInvoiceConnectionRequest,
	) (*pagination.CursorListResult[*invoice.Invoice], error)
	GetByID(
		ctx context.Context,
		req GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
	GetByIDs(
		ctx context.Context,
		req GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
	ListEmailAttempts(
		ctx context.Context,
		req ListInvoiceEmailAttemptsRequest,
	) (*pagination.ListResult[*invoice.EmailAttempt], error)
}

// InvoiceCreationService creates invoices from approved billing queue items, shipments and orders.
type InvoiceCreationService interface {
	CreateFromApprovedBillingQueueItem(
		ctx context.Context,
		req *CreateInvoiceFromBillingQueueRequest,
		actor *permissionports.RequestActor,
	) (*CreateInvoiceFromBillingQueueResult, error)
	CreateFromShipments(
		ctx context.Context,
		req *CreateInvoiceFromShipmentsRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	CreateFromOrder(
		ctx context.Context,
		req *CreateInvoiceFromOrderRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	CreateInvoicesFromShipments(
		ctx context.Context,
		req *CreateInvoiceFromShipmentsRequest,
		actor *permissionports.RequestActor,
	) (*CreateInvoicesResult, error)
	CreateInvoicesFromOrder(
		ctx context.Context,
		req *CreateInvoiceFromOrderRequest,
		actor *permissionports.RequestActor,
	) (*CreateInvoicesResult, error)
}

// InvoicePostingService posts invoices.
type InvoicePostingService interface {
	Post(
		ctx context.Context,
		req *PostInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	EnqueueAutoPost(
		ctx context.Context,
		entity *invoice.Invoice,
		actor *permissionports.RequestActor,
	) error
	PreviewPostSummary(
		ctx context.Context,
		req *PostInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*InvoicePostPreview, error)
	ReportAutoPostFailure(ctx context.Context, req *ReportAutoPostFailureRequest) error
}

// InvoicePDFService renders an invoice and generates and attaches its PDF.
type InvoicePDFService interface {
	RenderPreview(
		ctx context.Context,
		req *InvoicePreviewRequest,
	) (*InvoicePreviewResult, error)
	GeneratePDF(
		ctx context.Context,
		req *InvoicePreviewRequest,
		actor *permissionports.RequestActor,
	) (*GenerateInvoicePDFResult, error)
	AttachGeneratedPDF(
		ctx context.Context,
		req *AttachGeneratedPDFRequest,
	) (*invoice.Invoice, error)
}

// InvoiceSendService emails invoices and serves their shared documents.
type InvoiceSendService interface {
	PlanSend(
		ctx context.Context,
		req *InvoiceSendPlanRequest,
	) (*InvoiceSendPlan, error)
	Send(
		ctx context.Context,
		req *InvoiceSendRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceSendResult, error)
	SendFromWorkflow(
		ctx context.Context,
		req *InvoiceSendRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceSendResult, error)
	AutoSendInvoiceAfterPDFGeneration(
		ctx context.Context,
		req *AutoSendInvoiceAfterPDFGenerationRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceSendResult, error)
	DownloadSharedDocument(
		ctx context.Context,
		req *DownloadInvoiceDocumentRequest,
	) (*DownloadInvoiceDocumentResult, error)
}

// InvoiceEDIService resolves and sends invoices' outbound 210s.
type InvoiceEDIService interface {
	ResolveEDISendPlans(
		ctx context.Context,
		req *ResolveInvoiceEDISendPlansRequest,
	) (map[pulid.ID]*InvoiceEDISendPlan, error)
	SendEDI(
		ctx context.Context,
		req *SendInvoiceEDIRequest,
		actor *permissionports.RequestActor,
	) (*InvoiceEDISendResult, error)
}

// InvoiceMemoService raises credit and debit memos.
type InvoiceMemoService interface {
	CreateMemo(
		ctx context.Context,
		req *CreateMemoRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
}

// InvoiceVoidService voids invoices and discards drafts sent back for review.
type InvoiceVoidService interface {
	VoidInvoice(
		ctx context.Context,
		req *VoidInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*VoidInvoiceResult, error)
	DiscardDraftForReview(
		ctx context.Context,
		req *DiscardDraftForReviewRequest,
		actor *permissionports.RequestActor,
	) error
}

// InvoiceDraftService edits a draft invoice's delivery details.
type InvoiceDraftService interface {
	UpdateDraft(
		ctx context.Context,
		req *UpdateInvoiceDraftRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
}

// InvoicePostPreview is what posting an invoice would do, for the person
// about to post it: the ledger entry it books, the records it moves, where
// it goes next, or why it would be refused.
type InvoicePostPreview struct {
	InvoiceID    pulid.ID `json:"invoiceId"`
	Number       string   `json:"number"`
	BillToName   string   `json:"billToName"`
	CurrencyCode string   `json:"currencyCode"`
	TotalMinor   int64    `json:"totalMinor"`
	// Refusals are why Post would refuse the invoice as it stands; nothing
	// else is planned when there are any.
	Refusals      []string `json:"refusals"`
	AlreadyPosted bool     `json:"alreadyPosted"`
	// Journal is nil when the organization's accounting control books no
	// entry for this invoice.
	Journal         *InvoicePostJournal          `json:"journal"`
	ShipmentsBilled []string                     `json:"shipmentsBilled"`
	QueueItemPosted bool                         `json:"queueItemPosted"`
	AccountingSync  []InvoicePostSyncDestination `json:"accountingSync"`
	EDI             *InvoicePostEDI              `json:"edi"`
}

type InvoicePostJournal struct {
	AccountingDate   int64                    `json:"accountingDate"`
	EntryStatus      string                   `json:"entryStatus"`
	RequiresApproval bool                     `json:"requiresApproval"`
	Lines            []InvoicePostJournalLine `json:"lines"`
}

type InvoicePostJournalLine struct {
	GLAccountID pulid.ID `json:"glAccountId"`
	AccountCode string   `json:"accountCode"`
	AccountName string   `json:"accountName"`
	Description string   `json:"description"`
	DebitMinor  int64    `json:"debitMinor"`
	CreditMinor int64    `json:"creditMinor"`
}

type InvoicePostSyncDestination struct {
	Integration   string `json:"integration"`
	Company       string `json:"company"`
	AwaitsRelease bool   `json:"awaitsRelease"`
}

// InvoicePostEDI is the outbound 210 posting would queue; WillSend is false
// when the customer sends 210s by hand or a blocker stands in the way.
type InvoicePostEDI struct {
	PartnerName string   `json:"partnerName"`
	WillSend    bool     `json:"willSend"`
	Blockers    []string `json:"blockers"`
}

// InvoiceEDISendPlan is whether an invoice's outbound 210 can go, where it
// would go, and where the last attempt stands.
type InvoiceEDISendPlan struct {
	InvoiceID pulid.ID `json:"invoiceId"`
	// Enabled is the customer's EDI invoicing switch; Blockers say why a send
	// still cannot happen when it is on.
	Enabled             bool                  `json:"enabled"`
	AutoSend            bool                  `json:"autoSend"`
	PartnerID           pulid.ID              `json:"partnerId"`
	PartnerName         string                `json:"partnerName"`
	LadingVersionID     pulid.ID              `json:"ladingVersionId"`
	CommunicationMethod string                `json:"communicationMethod"`
	Status              invoice.EDISendStatus `json:"status"`
	LastMessageID       pulid.ID              `json:"lastMessageId"`
	LastError           string                `json:"lastError"`
	SentAt              *int64                `json:"sentAt"`
	Blockers            []string              `json:"blockers"`
}

type ResolveInvoiceEDISendPlansRequest struct {
	TenantInfo pagination.TenantInfo
	Invoices   []*invoice.Invoice
}

type SendInvoiceEDIRequest struct {
	InvoiceID  pulid.ID              `json:"invoiceId"`
	TenantInfo pagination.TenantInfo `json:"-"`
	// Force resends an invoice that already went out.
	Force bool `json:"force"`
}

type InvoiceEDISendResult struct {
	InvoiceID     pulid.ID              `json:"invoiceId"`
	Status        invoice.EDISendStatus `json:"status"`
	WorkflowID    string                `json:"workflowId"`
	WorkflowRunID string                `json:"workflowRunId"`
}

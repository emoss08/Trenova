package invoice

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Invoice)(nil)
	_ validationframework.TenantedEntity = (*Invoice)(nil)
	_ domaintypes.PostgresSearchable     = (*Invoice)(nil)
	_ bun.BeforeAppendModelHook          = (*InoviceLine)(nil)
	_ bun.BeforeAppendModelHook          = (*Attachment)(nil)
	_ bun.BeforeAppendModelHook          = (*EmailAttempt)(nil)
	_ bun.BeforeAppendModelHook          = (*EmailAttemptAttachment)(nil)
	_ bun.BeforeAppendModelHook          = (*DocumentShareToken)(nil)
)

type Invoice struct {
	bun.BaseModel `bun:"table:invoices,alias:inv" json:"-"`

	ID                        pulid.ID              `json:"id"                        bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID            pulid.ID              `json:"organizationId"            bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID            pulid.ID              `json:"businessUnitId"            bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	BillingQueueItemID        pulid.ID              `json:"billingQueueItemId"        bun:"billing_queue_item_id,type:VARCHAR(100),notnull"`
	ShipmentID                pulid.ID              `json:"shipmentId"                bun:"shipment_id,type:VARCHAR(100),notnull"`
	CustomerID                pulid.ID              `json:"customerId"                bun:"customer_id,type:VARCHAR(100),notnull"`
	Number                    string                `json:"number"                    bun:"number,type:VARCHAR(100),notnull"`
	BillType                  billingqueue.BillType `json:"billType"                  bun:"bill_type,type:VARCHAR(50),notnull"`
	Status                    Status                `json:"status"                    bun:"status,type:VARCHAR(50),notnull,default:'Draft'"`
	PaymentTerm               PaymentTerm           `json:"paymentTerm"               bun:"payment_term,type:VARCHAR(50),notnull"`
	CurrencyCode              string                `json:"currencyCode"              bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	InvoiceDate               int64                 `json:"invoiceDate"               bun:"invoice_date,type:BIGINT,notnull"`
	DueDate                   *int64                `json:"dueDate"                   bun:"due_date,type:BIGINT,nullzero"`
	PostedAt                  *int64                `json:"postedAt"                  bun:"posted_at,type:BIGINT,nullzero"`
	ShipmentProNumber         string                `json:"shipmentProNumber"         bun:"shipment_pro_number,type:VARCHAR(100),nullzero"`
	ShipmentBOL               string                `json:"shipmentBol"               bun:"shipment_bol,type:VARCHAR(100),nullzero"`
	ServiceDate               *int64                `json:"serviceDate"               bun:"service_date,type:BIGINT,nullzero"`
	BillToName                string                `json:"billToName"                bun:"bill_to_name,type:VARCHAR(255),notnull"`
	BillToCode                string                `json:"billToCode"                bun:"bill_to_code,type:VARCHAR(50),nullzero"`
	BillToAddressLine1        string                `json:"billToAddressLine1"        bun:"bill_to_address_line_1,type:VARCHAR(255),nullzero"`
	BillToAddressLine2        string                `json:"billToAddressLine2"        bun:"bill_to_address_line_2,type:VARCHAR(255),nullzero"`
	BillToCity                string                `json:"billToCity"                bun:"bill_to_city,type:VARCHAR(100),nullzero"`
	BillToState               string                `json:"billToState"               bun:"bill_to_state,type:VARCHAR(100),nullzero"`
	BillToPostalCode          string                `json:"billToPostalCode"          bun:"bill_to_postal_code,type:VARCHAR(20),nullzero"`
	BillToCountry             string                `json:"billToCountry"             bun:"bill_to_country,type:VARCHAR(100),nullzero"`
	SubtotalAmount            decimal.Decimal       `json:"subtotalAmount"            bun:"subtotal_amount,type:NUMERIC(19,4),notnull,default:0"`
	SubtotalAmountMinor       int64                 `json:"subtotalAmountMinor"       bun:"subtotal_amount_minor,type:BIGINT,notnull,default:0"`
	OtherAmount               decimal.Decimal       `json:"otherAmount"               bun:"other_amount,type:NUMERIC(19,4),notnull,default:0"`
	OtherAmountMinor          int64                 `json:"otherAmountMinor"          bun:"other_amount_minor,type:BIGINT,notnull,default:0"`
	TotalAmount               decimal.Decimal       `json:"totalAmount"               bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalAmountMinor          int64                 `json:"totalAmountMinor"          bun:"total_amount_minor,type:BIGINT,notnull,default:0"`
	AppliedAmount             decimal.Decimal       `json:"appliedAmount"             bun:"applied_amount,type:NUMERIC(19,4),notnull,default:0"`
	AppliedAmountMinor        int64                 `json:"appliedAmountMinor"        bun:"applied_amount_minor,type:BIGINT,notnull,default:0"`
	SettlementStatus          SettlementStatus      `json:"settlementStatus"          bun:"settlement_status,type:VARCHAR(50),notnull,default:'Unpaid'"`
	DisputeStatus             DisputeStatus         `json:"disputeStatus"             bun:"dispute_status,type:VARCHAR(50),notnull,default:'None'"`
	PDFDocumentID             pulid.ID              `json:"pdfDocumentId"             bun:"pdf_document_id,type:VARCHAR(100),nullzero"`
	SendStatus                SendStatus            `json:"sendStatus"                bun:"send_status,type:VARCHAR(50),notnull,default:'NotSent'"`
	SentAt                    *int64                `json:"sentAt"                    bun:"sent_at,type:BIGINT,nullzero"`
	SentByID                  pulid.ID              `json:"sentById"                  bun:"sent_by_id,type:VARCHAR(100),nullzero"`
	LastSendError             string                `json:"lastSendError"             bun:"last_send_error,type:TEXT,nullzero"`
	LastSendWarning           string                `json:"lastSendWarning"           bun:"last_send_warning,type:TEXT,nullzero"`
	Memo                      string                `json:"memo"                      bun:"memo,type:TEXT,nullzero"`
	RemittanceInstructions    string                `json:"remittanceInstructions"    bun:"remittance_instructions,type:TEXT,nullzero"`
	EmailSubjectSnapshot      string                `json:"emailSubjectSnapshot"      bun:"email_subject_snapshot,type:VARCHAR(998),nullzero"`
	EmailBodySnapshot         string                `json:"emailBodySnapshot"         bun:"email_body_snapshot,type:TEXT,nullzero"`
	EmailToSnapshot           []string              `json:"emailToSnapshot"           bun:"email_to_snapshot,array,type:text[],nullzero"`
	EmailCCSnapshot           []string              `json:"emailCcSnapshot"           bun:"email_cc_snapshot,array,type:text[],nullzero"`
	EmailBCCSnapshot          []string              `json:"emailBccSnapshot"          bun:"email_bcc_snapshot,array,type:text[],nullzero"`
	CorrectionGroupID         pulid.ID              `json:"correctionGroupId"         bun:"correction_group_id,type:VARCHAR(100),nullzero"`
	SupersedesInvoiceID       pulid.ID              `json:"supersedesInvoiceId"       bun:"supersedes_invoice_id,type:VARCHAR(100),nullzero"`
	SupersededByInvoiceID     pulid.ID              `json:"supersededByInvoiceId"     bun:"superseded_by_invoice_id,type:VARCHAR(100),nullzero"`
	SourceInvoiceAdjustmentID pulid.ID              `json:"sourceInvoiceAdjustmentId" bun:"source_invoice_adjustment_id,type:VARCHAR(100),nullzero"`
	IsAdjustmentArtifact      bool                  `json:"isAdjustmentArtifact"      bun:"is_adjustment_artifact,type:BOOLEAN,notnull,default:false"`
	Version                   int64                 `json:"version"                   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                 int64                 `json:"createdAt"                 bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                 int64                 `json:"updatedAt"                 bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BillingQueueItem *billingqueue.BillingQueueItem `json:"billingQueueItem,omitempty" bun:"rel:belongs-to,join:billing_queue_item_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Shipment         *shipment.Shipment             `json:"shipment,omitempty"         bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer         *customer.Customer             `json:"customer,omitempty"         bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PDFDocument      *document.Document             `json:"pdfDocument,omitempty"      bun:"rel:belongs-to,join:pdf_document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Lines            []*InoviceLine                 `json:"lines,omitempty"            bun:"rel:has-many,join:id=invoice_id"`
	Attachments      []*Attachment                  `json:"attachments,omitempty"      bun:"rel:has-many,join:id=invoice_id"`
	EmailAttempts    []*EmailAttempt                `json:"emailAttempts,omitempty"    bun:"rel:has-many,join:id=invoice_id"`
}

type InoviceLine struct {
	bun.BaseModel `bun:"table:invoice_lines,alias:invl" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	LineNumber     int             `json:"lineNumber"     bun:"line_number,type:INTEGER,notnull"`
	Type           InvoiceLineType `json:"type"           bun:"type,type:VARCHAR(50),notnull"`
	Description    string          `json:"description"    bun:"description,type:TEXT,notnull"`
	Quantity       decimal.Decimal `json:"quantity"       bun:"quantity,type:NUMERIC(19,4),notnull,default:0"`
	UnitPrice      decimal.Decimal `json:"unitPrice"      bun:"unit_price,type:NUMERIC(19,4),notnull,default:0"`
	Amount         decimal.Decimal `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	AmountMinor    int64           `json:"amountMinor"    bun:"amount_minor,type:BIGINT,notnull,default:0"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice *Invoice `json:"-" bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type Attachment struct {
	bun.BaseModel `bun:"table:invoice_attachments,alias:inva" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	DocumentID     pulid.ID `json:"documentId"     bun:"document_id,type:VARCHAR(100),notnull"`
	Selected       bool     `json:"selected"       bun:"selected,type:BOOLEAN,notnull,default:true"`
	SortOrder      int      `json:"sortOrder"      bun:"sort_order,type:INTEGER,notnull,default:0"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice  *Invoice           `json:"-"                  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document *document.Document `json:"document,omitempty" bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type EmailAttempt struct {
	bun.BaseModel `bun:"table:invoice_email_attempts,alias:inea" json:"-"`

	ID                pulid.ID       `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID         pulid.ID       `json:"invoiceId"         bun:"invoice_id,type:VARCHAR(100),notnull"`
	EmailMessageID    pulid.ID       `json:"emailMessageId"    bun:"email_message_id,type:VARCHAR(100),nullzero"`
	AttemptNumber     int            `json:"attemptNumber"     bun:"attempt_number,type:INTEGER,notnull"`
	PartNumber        int            `json:"partNumber"        bun:"part_number,type:INTEGER,notnull"`
	TotalParts        int            `json:"totalParts"        bun:"total_parts,type:INTEGER,notnull"`
	Status            SendStatus     `json:"status"            bun:"status,type:VARCHAR(50),notnull"`
	Provider          email.Provider `json:"provider"          bun:"provider,type:VARCHAR(50),nullzero"`
	ProviderMessageID string         `json:"providerMessageId" bun:"provider_message_id,type:VARCHAR(160),nullzero"`
	ToRecipients      []string       `json:"toRecipients"      bun:"to_recipients,array,type:text[],notnull"`
	CCRecipients      []string       `json:"ccRecipients"      bun:"cc_recipients,array,type:text[],nullzero"`
	BCCRecipients     []string       `json:"bccRecipients"     bun:"bcc_recipients,array,type:text[],nullzero"`
	Subject           string         `json:"subject"           bun:"subject,type:VARCHAR(998),notnull"`
	Body              string         `json:"body"              bun:"body,type:TEXT,nullzero"`
	EstimatedSize     int64          `json:"estimatedSize"     bun:"estimated_size,type:BIGINT,notnull,default:0"`
	Warnings          []string       `json:"warnings"          bun:"warnings,array,type:text[],nullzero"`
	Error             string         `json:"error"             bun:"error,type:TEXT,nullzero"`
	SentAt            *int64         `json:"sentAt"            bun:"sent_at,type:BIGINT,nullzero"`
	CreatedByID       pulid.ID       `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64          `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice     *Invoice                  `json:"-"                     bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Email       *email.Message            `json:"email,omitempty"        bun:"rel:belongs-to,join:email_message_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Attachments []*EmailAttemptAttachment `json:"attachments,omitempty"  bun:"rel:has-many,join:id=attempt_id"`
}

type EmailAttemptAttachment struct {
	bun.BaseModel `bun:"table:invoice_email_attempt_attachments,alias:ineaa" json:"-"`

	ID             pulid.ID                 `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID                 `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AttemptID      pulid.ID                 `json:"attemptId"      bun:"attempt_id,type:VARCHAR(100),notnull"`
	DocumentID     pulid.ID                 `json:"documentId"     bun:"document_id,type:VARCHAR(100),notnull"`
	FileName       string                   `json:"fileName"       bun:"file_name,type:VARCHAR(255),notnull"`
	ContentType    string                   `json:"contentType"    bun:"content_type,type:VARCHAR(120),notnull"`
	SizeBytes      int64                    `json:"sizeBytes"      bun:"size_bytes,type:BIGINT,notnull"`
	EncodedBytes   int64                    `json:"encodedBytes"   bun:"encoded_bytes,type:BIGINT,notnull"`
	Method         AttachmentDeliveryMethod `json:"method"         bun:"method,type:VARCHAR(50),notnull"`
	ShareTokenID   pulid.ID                 `json:"shareTokenId"   bun:"share_token_id,type:VARCHAR(100),nullzero"`
	Reason         string                   `json:"reason"         bun:"reason,type:TEXT,nullzero"`
	CreatedAt      int64                    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Attempt    *EmailAttempt       `json:"-"                    bun:"rel:belongs-to,join:attempt_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document   *document.Document  `json:"document,omitempty"   bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	ShareToken *DocumentShareToken `json:"shareToken,omitempty" bun:"rel:belongs-to,join:share_token_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type DocumentShareToken struct {
	bun.BaseModel `bun:"table:invoice_document_share_tokens,alias:indst" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	DocumentID     pulid.ID `json:"documentId"     bun:"document_id,type:VARCHAR(100),notnull"`
	TokenHash      string   `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull"`
	ExpiresAt      int64    `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	DownloadedAt   *int64   `json:"downloadedAt"   bun:"downloaded_at,type:BIGINT,nullzero"`
	RevokedAt      *int64   `json:"revokedAt"      bun:"revoked_at,type:BIGINT,nullzero"`
	CreatedByID    pulid.ID `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice      *Invoice             `json:"-"                  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document     *document.Document   `json:"document,omitempty" bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Organization *tenant.Organization `json:"-"                  bun:"rel:belongs-to,join:organization_id=id"`
}

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (i *Invoice) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		i,
		validation.Field(
			&i.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&i.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&i.BillingQueueItemID,
			validation.Required.Error("Billing queue item ID is required"),
		),
		validation.Field(&i.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&i.CustomerID, validation.Required.Error("Customer ID is required")),
		validation.Field(&i.Number,
			validation.Required.Error("Invoice number is required"),
			validation.Length(1, 100).Error("Invoice number must be between 1 and 100 characters"),
		),
		validation.Field(&i.BillType,
			validation.Required.Error("Bill type is required"),
			validation.In(
				billingqueue.BillTypeInvoice,
				billingqueue.BillTypeCreditMemo,
				billingqueue.BillTypeDebitMemo,
			).Error("Invalid bill type"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Invoice status is required"),
			validation.By(func(value any) error {
				status, _ := value.(Status)
				if !status.IsValid() {
					return errors.New("invalid invoice status")
				}
				return nil
			}),
		),
		validation.Field(&i.PaymentTerm,
			validation.Required.Error("Payment term is required"),
			validation.By(func(value any) error {
				term, _ := value.(PaymentTerm)
				if !term.IsValid() {
					return errors.New("invalid payment term")
				}
				return nil
			}),
		),
		validation.Field(&i.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be a 3-character ISO code"),
		),
		validation.Field(&i.InvoiceDate, validation.Required.Error("Invoice date is required")),
		validation.Field(&i.SettlementStatus,
			validation.By(func(value any) error {
				status, _ := value.(SettlementStatus)
				if !status.IsValid() {
					return errors.New("invalid settlement status")
				}
				return nil
			}),
		),
		validation.Field(&i.DisputeStatus,
			validation.By(func(value any) error {
				status, _ := value.(DisputeStatus)
				if !status.IsValid() {
					return errors.New("invalid dispute status")
				}
				return nil
			}),
		),
		validation.Field(&i.SendStatus,
			validation.By(func(value any) error {
				status, _ := value.(SendStatus)
				if status == "" {
					return nil
				}
				if !status.IsValid() {
					return errors.New("invalid invoice send status")
				}
				return nil
			}),
		),
		validation.Field(&i.BillToName,
			validation.Required.Error("Bill-to name is required"),
			validation.Length(1, 255).Error("Bill-to name must be between 1 and 255 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if i.BillType == billingqueue.BillTypeCreditMemo {
		if i.TotalAmount.GreaterThan(decimal.Zero) {
			multiErr.Add(
				"totalAmount",
				errortypes.ErrInvalid,
				"Credit memo total must be zero or negative",
			)
		}
	} else if i.TotalAmount.LessThan(decimal.Zero) {
		multiErr.Add("totalAmount", errortypes.ErrInvalid, "Invoice total must be zero or positive")
	}

	if i.AppliedAmount.IsNegative() {
		multiErr.Add("appliedAmount", errortypes.ErrInvalid, "Applied amount must not be negative")
	}

	if len(i.Lines) == 0 {
		multiErr.Add("lines", errortypes.ErrRequired, "At least one invoice line is required")
	}

	for idx, line := range i.Lines {
		if line == nil {
			multiErr.Add(
				"lines",
				errortypes.ErrInvalid,
				"Invoice lines must not contain null values",
			)
			continue
		}

		line.Validate(multiErr, idx)
	}
}

func (i *Invoice) OpenBalanceAmount() decimal.Decimal {
	if i.BillType == billingqueue.BillTypeCreditMemo {
		return decimal.Zero
	}

	openBalance := i.TotalAmount.Sub(i.AppliedAmount)
	if openBalance.IsNegative() {
		return decimal.Zero
	}

	return openBalance
}

func (i *Invoice) SyncMinorAmounts() {
	i.SubtotalAmountMinor = money.MinorUnits(i.SubtotalAmount)
	i.OtherAmountMinor = money.MinorUnits(i.OtherAmount)
	i.TotalAmountMinor = money.MinorUnits(i.TotalAmount)
	i.AppliedAmountMinor = money.MinorUnits(i.AppliedAmount)

	for _, line := range i.Lines {
		if line == nil {
			continue
		}

		line.SyncMinorAmount()
	}
}

func (i *Invoice) OpenBalanceMinor() int64 {
	if i.BillType == billingqueue.BillTypeCreditMemo {
		return 0
	}
	openBalance := i.TotalAmountMinor - i.AppliedAmountMinor
	if openBalance < 0 {
		return 0
	}
	return openBalance
}

func (i *Invoice) ApplyPaymentMinor(amountMinor int64) {
	if amountMinor <= 0 {
		return
	}
	i.AppliedAmountMinor += amountMinor
	i.AppliedAmount = money.DecimalFromMinor(i.AppliedAmountMinor)
	switch {
	case i.AppliedAmountMinor <= 0:
		i.SettlementStatus = SettlementStatusUnpaid
	case i.AppliedAmountMinor >= i.TotalAmountMinor:
		i.SettlementStatus = SettlementStatusPaid
	default:
		i.SettlementStatus = SettlementStatusPartiallyPaid
	}
}

func (i *Invoice) RemovePaymentMinor(amountMinor int64) {
	if amountMinor <= 0 {
		return
	}
	i.AppliedAmountMinor -= amountMinor
	if i.AppliedAmountMinor < 0 {
		i.AppliedAmountMinor = 0
	}
	i.AppliedAmount = money.DecimalFromMinor(i.AppliedAmountMinor)
	switch {
	case i.AppliedAmountMinor <= 0:
		i.SettlementStatus = SettlementStatusUnpaid
	case i.AppliedAmountMinor >= i.TotalAmountMinor:
		i.SettlementStatus = SettlementStatusPaid
	default:
		i.SettlementStatus = SettlementStatusPartiallyPaid
	}
}

func (l *InoviceLine) SyncMinorAmount() {
	l.AmountMinor = money.MinorUnits(l.Amount)
}

func (l *InoviceLine) Validate(multiErr *errortypes.MultiError, idx int) {
	prefix := "lines[" + strconv.Itoa(idx) + "]"

	if l.Type == "" || !l.Type.IsValid() {
		multiErr.Add(prefix+".type", errortypes.ErrInvalid, "Invalid invoice line type")
	}
	if strings.TrimSpace(l.Description) == "" {
		multiErr.Add(
			prefix+".description",
			errortypes.ErrRequired,
			"Invoice line description is required",
		)
	}
	if l.Quantity.LessThanOrEqual(decimal.Zero) {
		multiErr.Add(
			prefix+".quantity",
			errortypes.ErrInvalid,
			"Invoice line quantity must be greater than zero",
		)
	}
}

func (i *Invoice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("inv_")
		}
		if i.SendStatus == "" {
			i.SendStatus = SendStatusNotSent
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (a *Attachment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("invatt_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}

func (a *EmailAttempt) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("invea_")
		}
		if a.Status == "" {
			a.Status = SendStatusSending
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}

func (a *EmailAttemptAttachment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("inveaa_")
		}
		a.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

func (t *DocumentShareToken) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("invdst_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}
	return nil
}

func (l *InoviceLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("invl_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}

func (i *Invoice) GetID() pulid.ID {
	return i.ID
}

func (i *Invoice) GetTableName() string {
	return "invoices"
}

func (i *Invoice) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *Invoice) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *Invoice) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "inv",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "number", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{
				Name:   "bill_to_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "shipment_pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "shipment_bol",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "shipment",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*shipment.Shipment)(nil),
				TargetTable:  "shipments",
				ForeignKey:   "shipment_id",
				ReferenceKey: "id",
				Alias:        "sp",
				Queryable:    true,
			},
			{
				Field:        "customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cus",
				Queryable:    true,
			},
		},
	}
}

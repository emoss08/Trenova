package invoice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*Invoice)(nil)
	_ domaintypes.PostgresSearchable     = (*Invoice)(nil)
)

// Customer invoices
type Invoice struct {
	keel.Table `keel:"invoices,report=Billing"`

	ID                        pulid.ID                   `json:"id"                        keel:"id,pk,autoid:inv_"`
	OrganizationID            pulid.ID                   `json:"organizationId"            keel:"organization_id,pk,tenant"`
	BusinessUnitID            pulid.ID                   `json:"businessUnitId"            keel:"business_unit_id,pk,tenant"`
	BillingQueueItemID        pulid.ID                   `json:"billingQueueItemId"        keel:"billing_queue_item_id"`
	ShipmentID                pulid.ID                   `json:"shipmentId"                keel:"shipment_id,nullzero"`
	OrderID                   pulid.ID                   `json:"orderId"                   keel:"order_id,nullzero"`
	CustomerID                pulid.ID                   `json:"customerId"                keel:"customer_id"`
	Scope                     Scope                      `json:"scope"                     keel:"scope"`
	InvoiceRunID              pulid.ID                   `json:"invoiceRunId"              keel:"invoice_run_id,nullzero"`
	PeriodStart               *int64                     `json:"periodStart"               keel:"period_start,type=epoch"`
	PeriodEnd                 *int64                     `json:"periodEnd"                 keel:"period_end,type=epoch"`
	ShipmentCount             int                        `json:"shipmentCount"             keel:"shipment_count"`
	Detail                    customer.InvoiceDetail     `json:"detail"                    keel:"detail"`
	SectionBy                 customer.InvoiceSectionKey `json:"sectionBy"                 keel:"section_by"`
	OffCycleReason            string                     `json:"offCycleReason"            keel:"off_cycle_reason,nullzero"`
	Number                    string                     `json:"number"                    keel:"number,label=Invoice Number"`
	BillType                  billingqueue.BillType      `json:"billType"                  keel:"bill_type"`
	Status                    Status                     `json:"status"                    keel:"status"`
	PaymentTerm               PaymentTerm                `json:"paymentTerm"               keel:"payment_term"`
	CurrencyCode              string                     `json:"currencyCode"              keel:"currency_code"`
	InvoiceDate               int64                      `json:"invoiceDate"               keel:"invoice_date"`
	DueDate                   *int64                     `json:"dueDate"                   keel:"due_date"`
	PostedAt                  *int64                     `json:"postedAt"                  keel:"posted_at"`
	ExchangeRate              decimal.NullDecimal        `json:"exchangeRate"              keel:"exchange_rate"`
	ExchangeRateDate          *int64                     `json:"exchangeRateDate"          keel:"exchange_rate_date"`
	ShipmentProNumber         string                     `json:"shipmentProNumber"         keel:"shipment_pro_number,nullzero"`
	ShipmentBOL               string                     `json:"shipmentBol"               keel:"shipment_bol,nullzero"`
	OrderNumber               string                     `json:"orderNumber"               keel:"order_number,nullzero"`
	ServiceDate               *int64                     `json:"serviceDate"               keel:"service_date"`
	BillToName                string                     `json:"billToName"                keel:"bill_to_name"`
	BillToCode                string                     `json:"billToCode"                keel:"bill_to_code,nullzero"`
	BillToAddressLine1        string                     `json:"billToAddressLine1"        keel:"bill_to_address_line_1,nullzero"`
	BillToAddressLine2        string                     `json:"billToAddressLine2"        keel:"bill_to_address_line_2,nullzero"`
	BillToCity                string                     `json:"billToCity"                keel:"bill_to_city,nullzero"`
	BillToState               string                     `json:"billToState"               keel:"bill_to_state,nullzero"`
	BillToPostalCode          string                     `json:"billToPostalCode"          keel:"bill_to_postal_code,nullzero"`
	BillToCountry             string                     `json:"billToCountry"             keel:"bill_to_country,nullzero"`
	SubtotalAmount            decimal.Decimal            `json:"subtotalAmount"            keel:"subtotal_amount,label=Subtotal,format=money"`
	SubtotalAmountMinor       int64                      `json:"subtotalAmountMinor"       keel:"subtotal_amount_minor"`
	OtherAmount               decimal.Decimal            `json:"otherAmount"               keel:"other_amount,format=money"`
	OtherAmountMinor          int64                      `json:"otherAmountMinor"          keel:"other_amount_minor"`
	TotalAmount               decimal.Decimal            `json:"totalAmount"               keel:"total_amount,format=money"`
	TotalAmountMinor          int64                      `json:"totalAmountMinor"          keel:"total_amount_minor"`
	AppliedAmount             decimal.Decimal            `json:"appliedAmount"             keel:"applied_amount,format=money"`
	AppliedAmountMinor        int64                      `json:"appliedAmountMinor"        keel:"applied_amount_minor"`
	SettlementStatus          SettlementStatus           `json:"settlementStatus"          keel:"settlement_status"`
	DisputeStatus             DisputeStatus              `json:"disputeStatus"             keel:"dispute_status"`
	PDFDocumentID             pulid.ID                   `json:"pdfDocumentId"             keel:"pdf_document_id,nullzero"`
	SendStatus                SendStatus                 `json:"sendStatus"                keel:"send_status"`
	SentAt                    *int64                     `json:"sentAt"                    keel:"sent_at"`
	SentByID                  pulid.ID                   `json:"sentById"                  keel:"sent_by_id,nullzero"`
	LastSendError             string                     `json:"lastSendError"             keel:"last_send_error,nullzero"`
	LastSendWarning           string                     `json:"lastSendWarning"           keel:"last_send_warning,nullzero"`
	Memo                      string                     `json:"memo"                      keel:"memo,nullzero"`
	RatingNotes               []string                   `json:"ratingNotes"               keel:"rating_notes,nullzero"`
	RemittanceInstructions    string                     `json:"remittanceInstructions"    keel:"remittance_instructions,nullzero"`
	EmailSubjectSnapshot      string                     `json:"emailSubjectSnapshot"      keel:"email_subject_snapshot,nullzero"`
	EmailBodySnapshot         string                     `json:"emailBodySnapshot"         keel:"email_body_snapshot,nullzero"`
	EmailToSnapshot           []string                   `json:"emailToSnapshot"           keel:"email_to_snapshot,nullzero"`
	EmailCCSnapshot           []string                   `json:"emailCcSnapshot"           keel:"email_cc_snapshot,nullzero"`
	EmailBCCSnapshot          []string                   `json:"emailBccSnapshot"          keel:"email_bcc_snapshot,nullzero"`
	CorrectionGroupID         pulid.ID                   `json:"correctionGroupId"         keel:"correction_group_id,nullzero"`
	SupersedesInvoiceID       pulid.ID                   `json:"supersedesInvoiceId"       keel:"supersedes_invoice_id,nullzero"`
	SupersededByInvoiceID     pulid.ID                   `json:"supersededByInvoiceId"     keel:"superseded_by_invoice_id,nullzero"`
	SourceInvoiceAdjustmentID pulid.ID                   `json:"sourceInvoiceAdjustmentId" keel:"source_invoice_adjustment_id,nullzero"`
	IsAdjustmentArtifact      bool                       `json:"isAdjustmentArtifact"      keel:"is_adjustment_artifact"`
	ShipperCustomerID         pulid.ID                   `json:"shipperCustomerId"         keel:"shipper_customer_id,nullzero"`
	FactoringCompanyID        *pulid.ID                  `json:"factoringCompanyId"        keel:"factoring_company_id"`
	FactoringAgreementID      *pulid.ID                  `json:"factoringAgreementId"      keel:"factoring_agreement_id"`
	FactoringOverride         FactoringOverride          `json:"factoringOverride"         keel:"factoring_override"`
	FactoringOverrideReason   string                     `json:"factoringOverrideReason"   keel:"factoring_override_reason,nullzero"`
	IsSplitBill               bool                       `json:"isSplitBill"               keel:"is_split_bill"`
	VoidedAt                  *int64                     `json:"voidedAt"                  keel:"voided_at"`
	VoidedByID                pulid.ID                   `json:"voidedById"                keel:"voided_by_id,nullzero"`
	VoidReason                string                     `json:"voidReason"                keel:"void_reason,nullzero"`
	VoidDisposition           VoidDisposition            `json:"voidDisposition"           keel:"void_disposition,nullzero"`
	VoidedByAdjustmentID      pulid.ID                   `json:"voidedByAdjustmentId"      keel:"voided_by_adjustment_id,nullzero"`
	ReferenceInvoiceID        pulid.ID                   `json:"referenceInvoiceId"        keel:"reference_invoice_id,nullzero"`
	MemoReason                string                     `json:"memoReason"                keel:"memo_reason,nullzero"`
	MemoKind                  MemoKind                   `json:"memoKind"                  keel:"memo_kind,nullzero"`
	EDISendStatus             EDISendStatus              `json:"ediSendStatus"             keel:"edi_send_status"`
	LastEDIMessageID          pulid.ID                   `json:"lastEdiMessageId"          keel:"last_edi_message_id,nullzero"`
	EDISentAt                 *int64                     `json:"ediSentAt"                 keel:"edi_sent_at"`
	LastEDIError              string                     `json:"lastEdiError"              keel:"last_edi_error,nullzero"`
	BalanceDueMinor           int64                      `json:"balanceDueMinor"           keel:"balance_due_minor,readonly,label=Balance Due (Minor Units)"`
	Version                   int64                      `json:"version"                   keel:"version,version"`
	CreatedAt                 int64                      `json:"createdAt"                 keel:"created_at,autocreate"`
	UpdatedAt                 int64                      `json:"updatedAt"                 keel:"updated_at,autoupdate"`

	BillingQueueItem *billingqueue.BillingQueueItem     `json:"billingQueueItem,omitempty" keel:"belongs:billing_queue_item_id,noreport"`
	Shipment         *shipment.Shipment                 `json:"shipment,omitempty"         keel:"belongs:shipment_id"`
	Order            *order.Order                       `json:"order,omitempty"            keel:"belongs:order_id"`
	Customer         *customer.Customer                 `json:"customer,omitempty"         keel:"belongs:customer_id"`
	ShipperCustomer  *customer.Customer                 `json:"shipperCustomer,omitempty"  keel:"belongs:shipper_customer_id"`
	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
	ReferenceInvoice *Invoice                           `json:"referenceInvoice,omitempty" keel:"belongs:reference_invoice_id"`
	PDFDocument      *document.Document                 `json:"pdfDocument,omitempty"      keel:"belongs:pdf_document_id,noreport"`
	Lines            []*InvoiceLine                     `json:"lines,omitempty"            keel:"many:invoice_id,label=Invoice Lines"`
	Attachments      []*Attachment                      `json:"attachments,omitempty"      keel:"many:invoice_id,noreport"`
	EmailAttempts    []*EmailAttempt                    `json:"emailAttempts,omitempty"    keel:"many:invoice_id,noreport"`
}

// Line items on an invoice
type InvoiceLine struct {
	keel.Table `keel:"invoice_lines,report=Billing"`

	ID                pulid.ID        `json:"id"                keel:"id,pk,autoid:invl_"`
	OrganizationID    pulid.ID        `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	InvoiceID         pulid.ID        `json:"invoiceId"         keel:"invoice_id"`
	ShipmentID        pulid.ID        `json:"shipmentId"        keel:"shipment_id,nullzero"`
	ShipmentProNumber string          `json:"shipmentProNumber" keel:"shipment_pro_number,nullzero"`
	ShipmentBOL       string          `json:"shipmentBol"       keel:"shipment_bol,nullzero"`
	LineNumber        int             `json:"lineNumber"        keel:"line_number,dimension"`
	Type              InvoiceLineType `json:"type"              keel:"type"`
	Description       string          `json:"description"       keel:"description"`
	Quantity          decimal.Decimal `json:"quantity"          keel:"quantity"`
	UnitPrice         decimal.Decimal `json:"unitPrice"         keel:"unit_price,format=money"`
	Amount            decimal.Decimal `json:"amount"            keel:"amount,format=money"`
	AmountMinor       int64           `json:"amountMinor"       keel:"amount_minor"`

	AccessorialChargeID pulid.ID                 `json:"accessorialChargeId" keel:"accessorial_charge_id,nullzero"`
	ChargeCode          string                   `json:"chargeCode"          keel:"charge_code,nullzero"`
	ChargeMethod        accessorialcharge.Method `json:"chargeMethod"        keel:"charge_method,nullzero"`
	RateUnit            string                   `json:"rateUnit"            keel:"rate_unit,nullzero"`
	Rate                decimal.NullDecimal      `json:"rate"                keel:"rate"`
	RateBasisAmount     decimal.NullDecimal      `json:"rateBasisAmount"     keel:"rate_basis_amount"`
	FormulaTemplateID   pulid.ID                 `json:"formulaTemplateId"   keel:"formula_template_id,nullzero"`
	FormulaTemplateName string                   `json:"formulaTemplateName" keel:"formula_template_name,nullzero"`
	AllocationPercent   decimal.NullDecimal      `json:"allocationPercent"   keel:"allocation_percent"`
	ChargeAllocationID  pulid.ID                 `json:"chargeAllocationId"  keel:"charge_allocation_id,nullzero"`
	AdditionalChargeID  pulid.ID                 `json:"additionalChargeId"  keel:"additional_charge_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Invoice *Invoice `json:"-" keel:"belongs:invoice_id"`
}

type Attachment struct {
	keel.Table `keel:"invoice_attachments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:invatt_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	InvoiceID      pulid.ID `json:"invoiceId"      keel:"invoice_id"`
	DocumentID     pulid.ID `json:"documentId"     keel:"document_id"`
	Selected       bool     `json:"selected"       keel:"selected"`
	SortOrder      int      `json:"sortOrder"      keel:"sort_order"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Invoice  *Invoice           `json:"-"                  keel:"belongs:invoice_id"`
	Document *document.Document `json:"document,omitempty" keel:"belongs:document_id"`
}

type EmailAttempt struct {
	keel.Table `keel:"invoice_email_attempts"`

	ID                pulid.ID       `json:"id"                keel:"id,pk,autoid:invea_"`
	OrganizationID    pulid.ID       `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	InvoiceID         pulid.ID       `json:"invoiceId"         keel:"invoice_id"`
	EmailMessageID    pulid.ID       `json:"emailMessageId"    keel:"email_message_id,nullzero"`
	AttemptNumber     int            `json:"attemptNumber"     keel:"attempt_number"`
	PartNumber        int            `json:"partNumber"        keel:"part_number"`
	TotalParts        int            `json:"totalParts"        keel:"total_parts"`
	Status            SendStatus     `json:"status"            keel:"status"`
	Provider          email.Provider `json:"provider"          keel:"provider,nullzero"`
	ProviderMessageID string         `json:"providerMessageId" keel:"provider_message_id,nullzero"`
	ToRecipients      []string       `json:"toRecipients"      keel:"to_recipients"`
	CCRecipients      []string       `json:"ccRecipients"      keel:"cc_recipients,nullzero"`
	BCCRecipients     []string       `json:"bccRecipients"     keel:"bcc_recipients,nullzero"`
	Subject           string         `json:"subject"           keel:"subject"`
	Body              string         `json:"body"              keel:"body,nullzero"`
	EstimatedSize     int64          `json:"estimatedSize"     keel:"estimated_size"`
	Warnings          []string       `json:"warnings"          keel:"warnings,nullzero"`
	Error             string         `json:"error"             keel:"error,nullzero"`
	SentAt            *int64         `json:"sentAt"            keel:"sent_at"`
	CreatedByID       pulid.ID       `json:"createdById"       keel:"created_by_id,nullzero"`
	CreatedAt         int64          `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64          `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Invoice     *Invoice                  `json:"-"                     keel:"belongs:invoice_id"`
	Email       *email.Message            `json:"email,omitempty"       keel:"belongs:email_message_id"`
	Attachments []*EmailAttemptAttachment `json:"attachments,omitempty" keel:"many:attempt_id"`
}

type EmailAttemptAttachment struct {
	keel.Table `keel:"invoice_email_attempt_attachments"`

	ID             pulid.ID                 `json:"id"             keel:"id,pk,autoid:inveaa_"`
	OrganizationID pulid.ID                 `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AttemptID      pulid.ID                 `json:"attemptId"      keel:"attempt_id"`
	DocumentID     pulid.ID                 `json:"documentId"     keel:"document_id"`
	FileName       string                   `json:"fileName"       keel:"file_name"`
	ContentType    string                   `json:"contentType"    keel:"content_type"`
	SizeBytes      int64                    `json:"sizeBytes"      keel:"size_bytes"`
	EncodedBytes   int64                    `json:"encodedBytes"   keel:"encoded_bytes"`
	Method         AttachmentDeliveryMethod `json:"method"         keel:"method"`
	ShareTokenID   pulid.ID                 `json:"shareTokenId"   keel:"share_token_id,nullzero"`
	Reason         string                   `json:"reason"         keel:"reason,nullzero"`
	CreatedAt      int64                    `json:"createdAt"      keel:"created_at,autocreate"`

	Attempt    *EmailAttempt       `json:"-"                    keel:"belongs:attempt_id"`
	Document   *document.Document  `json:"document,omitempty"   keel:"belongs:document_id"`
	ShareToken *DocumentShareToken `json:"shareToken,omitempty" keel:"belongs:share_token_id"`
}

type DocumentShareToken struct {
	keel.Table `keel:"invoice_document_share_tokens"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:invdst_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	InvoiceID      pulid.ID `json:"invoiceId"      keel:"invoice_id"`
	DocumentID     pulid.ID `json:"documentId"     keel:"document_id"`
	TokenHash      string   `json:"-"              keel:"token_hash"`
	ExpiresAt      int64    `json:"expiresAt"      keel:"expires_at"`
	DownloadedAt   *int64   `json:"downloadedAt"   keel:"downloaded_at"`
	RevokedAt      *int64   `json:"revokedAt"      keel:"revoked_at"`
	CreatedByID    pulid.ID `json:"createdById"    keel:"created_by_id,nullzero"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Invoice      *Invoice             `json:"-"                  keel:"belongs:invoice_id"`
	Document     *document.Document   `json:"document,omitempty" keel:"belongs:document_id"`
	Organization *tenant.Organization `json:"-"`
}

func (i *Invoice) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		i,
		validation.Field(
			&i.FactoringOverride,
			domainvalidation.ValidEnum[FactoringOverride]("Invalid factoring override"),
		),
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
			domainvalidation.ValidEnum[Status]("invalid invoice status"),
		),
		validation.Field(&i.Scope,
			validation.Required.Error("Invoice scope is required"),
			domainvalidation.ValidEnum[Scope]("invalid invoice scope"),
		),
		validation.Field(&i.PaymentTerm,
			validation.Required.Error("Payment term is required"),
			domainvalidation.ValidEnum[PaymentTerm]("invalid payment term"),
		),
		validation.Field(&i.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be a 3-character ISO code"),
		),
		validation.Field(&i.InvoiceDate, validation.Required.Error("Invoice date is required")),
		validation.Field(&i.SettlementStatus,
			domainvalidation.ValidEnum[SettlementStatus]("invalid settlement status"),
		),
		validation.Field(&i.DisputeStatus,
			domainvalidation.ValidEnum[DisputeStatus]("invalid dispute status"),
		),
		validation.Field(&i.SendStatus,
			domainvalidation.ValidEnum[SendStatus]("invalid invoice send status"),
		),
		validation.Field(&i.BillToName,
			validation.Required.Error("Bill-to name is required"),
			validation.Length(1, 255).Error("Bill-to name must be between 1 and 255 characters"),
		),
	))

	i.validateScope(multiErr)

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

func (i *Invoice) validateScope(multiErr *errortypes.MultiError) {
	switch i.Scope {
	case ScopeShipment:
		if i.ShipmentID.IsNil() {
			multiErr.Add(
				"shipmentId",
				errortypes.ErrRequired,
				"A shipment invoice must name the shipment it bills",
			)
		}
	case ScopeOrder:
		if i.OrderID.IsNil() {
			multiErr.Add(
				"orderId",
				errortypes.ErrRequired,
				"An order invoice must name the order it bills",
			)
		}
	case ScopeConsolidated:
		if len(i.LegShipmentIDs()) == 0 {
			multiErr.Add(
				"lines",
				errortypes.ErrRequired,
				"A consolidated invoice must bill at least one shipment",
			)
		}
		if i.PeriodStart == nil || i.PeriodEnd == nil {
			multiErr.Add(
				"periodStart",
				errortypes.ErrRequired,
				"A consolidated invoice must state the period it covers",
			)
		}
	case ScopeAdjustment:
		if i.CorrectionGroupID.IsNil() && i.SupersedesInvoiceID.IsNil() &&
			i.SourceInvoiceAdjustmentID.IsNil() {
			multiErr.Add(
				"correctionGroupId",
				errortypes.ErrRequired,
				"An adjustment invoice must belong to a correction chain",
			)
		}
	case ScopeMemo:
		if i.BillType != billingqueue.BillTypeCreditMemo &&
			i.BillType != billingqueue.BillTypeDebitMemo {
			multiErr.Add(
				"billType",
				errortypes.ErrInvalid,
				"A standalone memo must be a credit memo or a debit memo",
			)
		}
		if strings.TrimSpace(i.MemoReason) == "" {
			multiErr.Add("memoReason", errortypes.ErrRequired, "A memo needs a reason")
		}
		if !i.MemoKind.IsValid() {
			multiErr.Add("memoKind", errortypes.ErrInvalid, "Invalid memo kind")
		}
		if !i.ShipmentID.IsNil() || !i.OrderID.IsNil() {
			multiErr.Add(
				"shipmentId",
				errortypes.ErrInvalid,
				"A standalone memo cannot name a shipment or an order",
			)
		}
	}

	if i.Status == StatusVoided {
		if strings.TrimSpace(i.VoidReason) == "" {
			multiErr.Add("voidReason", errortypes.ErrRequired, "A voided invoice must say why")
		}
		if i.VoidedAt == nil {
			multiErr.Add("voidedAt", errortypes.ErrRequired, "A voided invoice must say when")
		}
		if !i.VoidDisposition.IsValid() {
			multiErr.Add("voidDisposition", errortypes.ErrInvalid, "Invalid void disposition")
		}
	}
}

// IsOpen reports whether the invoice still carries a balance a customer owes.
func (i *Invoice) IsOpen() bool {
	return i != nil && i.Status == StatusPosted &&
		(i.BillType == billingqueue.BillTypeInvoice ||
			i.BillType == billingqueue.BillTypeDebitMemo) &&
		i.OpenBalanceMinor() > 0
}

func (i *Invoice) IsCreditMemo() bool {
	return i != nil && i.BillType == billingqueue.BillTypeCreditMemo
}

// CreditTotalMinor is the credit a memo carries, as a positive figure.
func (i *Invoice) CreditTotalMinor() int64 {
	if !i.IsCreditMemo() {
		return 0
	}
	if i.TotalAmountMinor < 0 {
		return -i.TotalAmountMinor
	}

	return i.TotalAmountMinor
}

// CreditRemainingMinor is what is left of a posted credit memo to apply.
func (i *Invoice) CreditRemainingMinor() int64 {
	if !i.IsCreditMemo() || i.Status != StatusPosted {
		return 0
	}
	remaining := i.CreditTotalMinor() - i.AppliedAmountMinor
	if remaining < 0 {
		return 0
	}

	return remaining
}

// ApplyCreditMinor records part of a credit memo being used against an invoice.
func (i *Invoice) ApplyCreditMinor(amountMinor int64) {
	if amountMinor <= 0 || !i.IsCreditMemo() {
		return
	}
	i.AppliedAmountMinor += amountMinor
	i.AppliedAmount = money.DecimalFromMinor(i.AppliedAmountMinor)
	i.syncCreditSettlement()
}

func (i *Invoice) ReleaseCreditMinor(amountMinor int64) {
	if amountMinor <= 0 || !i.IsCreditMemo() {
		return
	}
	i.AppliedAmountMinor -= amountMinor
	if i.AppliedAmountMinor < 0 {
		i.AppliedAmountMinor = 0
	}
	i.AppliedAmount = money.DecimalFromMinor(i.AppliedAmountMinor)
	i.syncCreditSettlement()
}

func (i *Invoice) syncCreditSettlement() {
	switch {
	case i.AppliedAmountMinor <= 0:
		i.SettlementStatus = SettlementStatusUnpaid
	case i.AppliedAmountMinor >= i.CreditTotalMinor():
		i.SettlementStatus = SettlementStatusPaid
	default:
		i.SettlementStatus = SettlementStatusPartiallyPaid
	}
}

// LegShipmentIDs returns the distinct shipments billed by the invoice, preferring the
// per-line attribution (which covers grouped invoices) and falling back to the header
// shipment id for legacy single-shipment invoices whose lines predate line-level
// attribution.
func (i *Invoice) LegShipmentIDs() []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(i.Lines))
	ids := make([]pulid.ID, 0, len(i.Lines))
	for _, line := range i.Lines {
		if line == nil || line.ShipmentID.IsNil() {
			continue
		}
		if _, ok := seen[line.ShipmentID]; ok {
			continue
		}
		seen[line.ShipmentID] = struct{}{}
		ids = append(ids, line.ShipmentID)
	}

	if len(ids) == 0 && !i.ShipmentID.IsNil() {
		ids = append(ids, i.ShipmentID)
	}

	return ids
}

func (i *Invoice) OpenBalanceAmount() decimal.Decimal {
	if i.BillType == billingqueue.BillTypeCreditMemo || i.Status == StatusVoided {
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
	if i.BillType == billingqueue.BillTypeCreditMemo || i.Status == StatusVoided {
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

func (l *InvoiceLine) CopyChargeDetail(src *InvoiceLine) {
	if src == nil {
		return
	}
	l.AccessorialChargeID = src.AccessorialChargeID
	l.ChargeCode = src.ChargeCode
	l.ChargeMethod = src.ChargeMethod
	l.RateUnit = src.RateUnit
	l.Rate = src.Rate
	l.RateBasisAmount = src.RateBasisAmount
	l.FormulaTemplateID = src.FormulaTemplateID
	l.FormulaTemplateName = src.FormulaTemplateName
	l.AllocationPercent = src.AllocationPercent
	l.ChargeAllocationID = src.ChargeAllocationID
	l.AdditionalChargeID = src.AdditionalChargeID
}

// IsPartialShare reports whether the line bills only part of its charge, which
// is what a split invoice says on the face of every shared line.
func (l *InvoiceLine) IsPartialShare() bool {
	return l != nil && l.AllocationPercent.Valid &&
		!l.AllocationPercent.Decimal.Equal(decimal.NewFromInt(100))
}

func (l *InvoiceLine) SyncMinorAmount() {
	l.AmountMinor = money.MinorUnits(l.Amount)
}

func (l *InvoiceLine) Validate(multiErr *errortypes.MultiError, idx int) {
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

func (i *Invoice) BeforeCreate(ctx context.Context) error {
	if i.Scope == "" {
		i.Scope = ScopeShipment
	}
	if i.Detail == "" {
		i.Detail = customer.InvoiceDetailDetailed
	}
	if i.SectionBy == "" {
		i.SectionBy = customer.InvoiceSectionKeyShipment
	}
	if i.Status == "" {
		i.Status = StatusDraft
	}
	if i.CurrencyCode == "" {
		i.CurrencyCode = "USD"
	}
	if i.SettlementStatus == "" {
		i.SettlementStatus = SettlementStatusUnpaid
	}
	if i.DisputeStatus == "" {
		i.DisputeStatus = DisputeStatusNone
	}
	if i.SendStatus == "" {
		i.SendStatus = SendStatusNotSent
	}
	if i.FactoringOverride == "" {
		i.FactoringOverride = FactoringOverrideDefault
	}
	if i.EDISendStatus == "" {
		i.EDISendStatus = EDISendStatusNotSent
	}

	return nil
}

func (i *Invoice) BeforeUpdate(ctx context.Context) error {
	if i.FactoringOverride == "" {
		i.FactoringOverride = FactoringOverrideDefault
	}

	return nil
}

func (a *EmailAttempt) BeforeCreate(ctx context.Context) error {
	if a.Status == "" {
		a.Status = SendStatusSending
	}

	return nil
}

func (i *Invoice) GetID() pulid.ID {
	return i.ID
}

func (i *Invoice) GetCreatedAt() int64 {
	return i.CreatedAt
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
			{Field: "shipment", Queryable: true},
			{Field: "customer", Queryable: true},
			{Field: "shipperCustomer", Queryable: true},
		},
	}
}

func (a *Attachment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&a.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&a.SortOrder,
			validation.Min(0).Error("Sort order cannot be negative"),
		),
	))
}

func (a *EmailAttempt) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&a.AttemptNumber,
			validation.Required.Error("Attempt number is required"),
			validation.Min(1).Error("Attempt number must be at least one"),
		),
		// A large invoice is split across several messages, so a part outside
		// its own total would leave the customer with an incomplete set and no
		// way to tell.
		validation.Field(&a.TotalParts,
			validation.Required.Error("Total parts is required"),
			validation.Min(1).Error("Total parts must be at least one"),
		),
		validation.Field(&a.PartNumber,
			validation.Required.Error("Part number is required"),
			validation.Min(1).Error("Part number must be at least one"),
			validation.Max(a.TotalParts).Error("Part number cannot exceed the total parts"),
		),
		validation.Field(&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SendStatus]("Status is invalid"),
		),
		validation.Field(&a.Provider,
			domainvalidation.ValidEnum[email.Provider]("Provider is invalid"),
		),
		validation.Field(&a.ProviderMessageID,
			validation.Length(0, maxProviderMessageIDLength).
				Error("Provider message id cannot be longer than 160 characters"),
		),
		validation.Field(&a.ToRecipients,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&a.CCRecipients,
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&a.BCCRecipients,
			validation.Each(is.EmailFormat.Error("Recipient must be a valid email address")),
		),
		validation.Field(&a.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, MaxEmailSubjectLength).
				Error("Subject cannot be longer than 998 characters"),
		),
		validation.Field(&a.EstimatedSize,
			validation.Min(int64(0)).Error("Estimated size cannot be negative"),
		),
		validation.Field(&a.Error,
			validation.When(
				a.Status == SendStatusFailed,
				validation.Required.Error("A failed attempt must record why it failed"),
			),
		),
	))

	for i, attachment := range a.Attachments {
		if attachment == nil {
			continue
		}
		attachment.Validate(multiErr.WithIndex("attachments", i))
	}
}

func (a *EmailAttemptAttachment) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning attempt are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&a.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, maxAttachmentFileNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&a.ContentType,
			validation.Required.Error("Content type is required"),
			validation.Length(1, maxAttachmentContentTypeLength).
				Error("Content type cannot be longer than 120 characters"),
		),
		validation.Field(&a.SizeBytes,
			validation.Min(int64(0)).Error("Size cannot be negative"),
		),
		validation.Field(&a.EncodedBytes,
			validation.Min(int64(0)).Error("Encoded size cannot be negative"),
		),
		validation.Field(&a.Method,
			validation.Required.Error("Delivery method is required"),
			domainvalidation.ValidEnum[AttachmentDeliveryMethod]("Delivery method is invalid"),
		),
		// A link delivery with no token points the customer at nothing, and a
		// dropped attachment with no reason gives support nothing to explain.
		validation.Field(&a.ShareTokenID,
			validation.When(
				a.Method == AttachmentDeliveryMethodLink,
				validation.Required.Error("A linked attachment must name its share token"),
			),
		),
		validation.Field(&a.Reason,
			validation.When(
				a.Method == AttachmentDeliveryMethodSkipped ||
					a.Method == AttachmentDeliveryMethodFailed,
				validation.Required.Error(
					"An attachment that was not sent must record why",
				),
			),
		),
	))
}

func (t *DocumentShareToken) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&t.DocumentID, validation.Required.Error("Document is required")),
		// The token is only ever stored hashed, and it must expire: a share link
		// with no expiry is an unauthenticated download that never closes.
		validation.Field(&t.TokenHash,
			validation.Required.Error("Token hash is required"),
			validation.Length(1, maxShareTokenHashLength).
				Error("Token hash cannot be longer than 128 characters"),
		),
		validation.Field(&t.ExpiresAt,
			validation.Required.Error("Expiry is required"),
			validation.Min(int64(1)).Error("Expiry must be a valid timestamp"),
		),
	))
}

const (
	maxProviderMessageIDLength     = 160
	MaxEmailSubjectLength          = 998
	maxAttachmentFileNameLength    = 255
	maxAttachmentContentTypeLength = 120
	maxShareTokenHashLength        = 128
)

// BilledIn reports whether the invoice is in the given currency. A blank code
// on either side means the default currency, which is what a blank is saved as.
func (i *Invoice) BilledIn(currencyCode string) bool {
	return strings.EqualFold(currencyOrDefault(i.CurrencyCode), currencyOrDefault(currencyCode))
}

func currencyOrDefault(code string) string {
	if code = strings.TrimSpace(code); code == "" {
		return money.DefaultCurrencyCode
	}

	return code
}

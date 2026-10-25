package customerpayment

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// Cash received from customers and how much of it is applied
type Payment struct {
	keel.Table `keel:"customer_payments,report=Accounting,label=Customer Payment"`

	ID                   pulid.ID            `json:"id"                   keel:"id,pk,autoid:cpay_"`
	OrganizationID       pulid.ID            `json:"organizationId"       keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	CustomerID           pulid.ID            `json:"customerId"           keel:"customer_id"`
	PaymentDate          int64               `json:"paymentDate"          keel:"payment_date"`
	AccountingDate       int64               `json:"accountingDate"       keel:"accounting_date"`
	AmountMinor          int64               `json:"amountMinor"          keel:"amount_minor,label=Amount (Minor)"`
	AppliedAmountMinor   int64               `json:"appliedAmountMinor"   keel:"applied_amount_minor,label=Applied (Minor)"`
	UnappliedAmountMinor int64               `json:"unappliedAmountMinor" keel:"unapplied_amount_minor,label=Unapplied (Minor)"`
	Status               Status              `json:"status"               keel:"status"`
	PaymentMethod        Method              `json:"paymentMethod"        keel:"payment_method"`
	ReferenceNumber      string              `json:"referenceNumber"      keel:"reference_number,nullzero"`
	Memo                 string              `json:"memo"                 keel:"memo,nullzero"`
	CurrencyCode         string              `json:"currencyCode"         keel:"currency_code,label=Currency"`
	ExchangeRate         decimal.NullDecimal `json:"exchangeRate"         keel:"exchange_rate"`
	ExchangeRateDate     *int64              `json:"exchangeRateDate"     keel:"exchange_rate_date"`
	PostedBatchID        pulid.ID            `json:"postedBatchId"        keel:"posted_batch_id,nullzero"`
	ReversalBatchID      pulid.ID            `json:"reversalBatchId"      keel:"reversal_batch_id,nullzero"`
	ReversedByID         pulid.ID            `json:"reversedById"         keel:"reversed_by_id,nullzero"`
	ReversedAt           *int64              `json:"reversedAt"           keel:"reversed_at"`
	ReversalReason       string              `json:"reversalReason"       keel:"reversal_reason,nullzero"`
	CreatedByID          pulid.ID            `json:"createdById"          keel:"created_by_id"`
	UpdatedByID          pulid.ID            `json:"updatedById"          keel:"updated_by_id,nullzero"`
	Version              int64               `json:"version"              keel:"version,version"`
	CreatedAt            int64               `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt"            keel:"updated_at,autoupdate"`

	Applications []*Application     `json:"applications,omitempty" keel:"many:customer_payment_id"`
	Customer     *customer.Customer `json:"customer,omitempty"     keel:"belongs:customer_id"`
}

// Which invoice each dollar of a payment was applied to
type Application struct {
	keel.Table `keel:"customer_payment_applications,report=Accounting,label=Payment Application"`

	ID                  pulid.ID `json:"id"                  keel:"id,pk,autoid:cpapp_"`
	OrganizationID      pulid.ID `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	CustomerPaymentID   pulid.ID `json:"customerPaymentId"   keel:"customer_payment_id"`
	InvoiceID           pulid.ID `json:"invoiceId"           keel:"invoice_id"`
	AppliedAmountMinor  int64    `json:"appliedAmountMinor"  keel:"applied_amount_minor,label=Applied (Minor)"`
	ShortPayAmountMinor int64    `json:"shortPayAmountMinor" keel:"short_pay_amount_minor,label=Short Pay (Minor)"`
	LineNumber          int      `json:"lineNumber"          keel:"line_number,label=Line,dimension"`
	CreatedAt           int64    `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64    `json:"updatedAt"           keel:"updated_at,autoupdate"`

	Payment *Payment         `json:"payment,omitempty" keel:"belongs:customer_payment_id,label=Customer Payment"`
	Invoice *invoice.Invoice `json:"invoice,omitempty" keel:"belongs:invoice_id"`
}

func (p *Payment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required),
		validation.Field(&p.BusinessUnitID, validation.Required),
		validation.Field(&p.CustomerID, validation.Required),
		validation.Field(&p.PaymentDate, validation.Required),
		validation.Field(&p.AccountingDate, validation.Required),
		validation.Field(&p.CurrencyCode, validation.Required, validation.Length(3, 3)),
	))

	if p.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Payment amount must be greater than zero",
		)
	}
	if !p.PaymentMethod.IsValid() {
		multiErr.Add("paymentMethod", errortypes.ErrInvalid, "Payment method is invalid")
	}
	for idx, app := range p.Applications {
		if app == nil {
			multiErr.Add(
				"applications",
				errortypes.ErrInvalid,
				"Payment applications must not contain null values",
			)
			continue
		}
		app.Validate(multiErr.WithIndex("applications", idx))
	}
}

func (a *Application) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.InvoiceID, validation.Required),
	))
	if a.AppliedAmountMinor < 0 {
		multiErr.Add(
			"appliedAmountMinor",
			errortypes.ErrInvalid,
			"Applied amount cannot be negative",
		)
	}
	if a.ShortPayAmountMinor < 0 {
		multiErr.Add(
			"shortPayAmountMinor",
			errortypes.ErrInvalid,
			"Short-pay amount cannot be negative",
		)
	}
	if a.AppliedAmountMinor+a.ShortPayAmountMinor <= 0 {
		multiErr.Add(
			"appliedAmountMinor",
			errortypes.ErrInvalid,
			"Applied amount plus short-pay amount must be greater than zero",
		)
	}
}

func (p *Payment) SyncAmounts() {
	var applied int64
	for idx, app := range p.Applications {
		if app == nil {
			continue
		}
		app.LineNumber = idx + 1
		applied += app.AppliedAmountMinor
	}
	p.AppliedAmountMinor = applied
	p.UnappliedAmountMinor = max(p.AmountMinor-applied, 0)
	if strings.TrimSpace(p.CurrencyCode) == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}
}

func (p *Payment) DocumentLabel() string {
	if ref := strings.TrimSpace(p.ReferenceNumber); ref != "" {
		return ref
	}
	return money.FormatMinor(p.AmountMinor, p.CurrencyCode) + " on " +
		time.Unix(p.PaymentDate, 0).UTC().Format("2006-01-02")
}

func (p *Payment) GetTableName() string {
	return "customer_payments"
}

func (p *Payment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "cp",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "reference_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "memo", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (p *Payment) GetID() pulid.ID {
	return p.ID
}

func (p *Payment) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (p *Payment) IsFullyApplied() bool {
	return p.UnappliedAmountMinor == 0
}

func (p *Payment) CanReverse() bool {
	return p != nil && p.Status == StatusPosted
}

func (p *Payment) BeforeCreate(ctx context.Context) error {
	if p.CurrencyCode == "" {
		p.CurrencyCode = "USD"
	}

	return nil
}

// SettledAmountMinor is how much of its invoice an application closes: the
// cash applied plus any short pay written off. Posting settles exactly this
// much, so reversing must take back exactly this much.
func (a *Application) SettledAmountMinor() int64 {
	return a.AppliedAmountMinor + a.ShortPayAmountMinor
}

// ShortPayAmountMinor is the short pay written off across every application.
func (p *Payment) ShortPayAmountMinor() int64 {
	var total int64
	for _, app := range p.Applications {
		if app != nil {
			total += app.ShortPayAmountMinor
		}
	}

	return total
}

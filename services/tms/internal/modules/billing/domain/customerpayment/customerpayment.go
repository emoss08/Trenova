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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// Cash received from customers and how much of it is applied
type Payment struct {
	bun.BaseModel             `bun:"table:customer_payments,alias:cp" json:"-"`
	keel.Table                `                                                keel:"customer_payments,report=Accounting,label=Customer Payment"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                   pulid.ID            `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:cpay_"`
	OrganizationID       pulid.ID            `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	CustomerID           pulid.ID            `json:"customerId"           bun:"customer_id,type:VARCHAR(100),notnull"                                                keel:"customer_id"`
	PaymentDate          int64               `json:"paymentDate"          bun:"payment_date,type:BIGINT,notnull"                                                     keel:"payment_date"`
	AccountingDate       int64               `json:"accountingDate"       bun:"accounting_date,type:BIGINT,notnull"                                                  keel:"accounting_date"`
	AmountMinor          int64               `json:"amountMinor"          bun:"amount_minor,type:BIGINT,notnull"                                                     keel:"amount_minor,label=Amount (Minor)"`
	AppliedAmountMinor   int64               `json:"appliedAmountMinor"   bun:"applied_amount_minor,type:BIGINT,notnull"                                             keel:"applied_amount_minor,label=Applied (Minor)"`
	UnappliedAmountMinor int64               `json:"unappliedAmountMinor" bun:"unapplied_amount_minor,type:BIGINT,notnull"                                           keel:"unapplied_amount_minor,label=Unapplied (Minor)"`
	Status               Status              `json:"status"               bun:"status,type:VARCHAR(50),notnull"                                                      keel:"status"`
	PaymentMethod        Method              `json:"paymentMethod"        bun:"payment_method,type:VARCHAR(50),notnull"                                              keel:"payment_method"`
	ReferenceNumber      string              `json:"referenceNumber"      bun:"reference_number,type:VARCHAR(100),nullzero"                                          keel:"reference_number,nullzero"`
	Memo                 string              `json:"memo"                 bun:"memo,type:TEXT,nullzero"                                                              keel:"memo,nullzero"`
	CurrencyCode         string              `json:"currencyCode"         bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"                                  keel:"currency_code,label=Currency"`
	ExchangeRate         decimal.NullDecimal `json:"exchangeRate"         bun:"exchange_rate,type:NUMERIC(24,12),nullzero"                                           keel:"exchange_rate"`
	ExchangeRateDate     *int64              `json:"exchangeRateDate"     bun:"exchange_rate_date,type:BIGINT,nullzero"                                              keel:"exchange_rate_date"`
	PostedBatchID        pulid.ID            `json:"postedBatchId"        bun:"posted_batch_id,type:VARCHAR(100),nullzero"                                           keel:"posted_batch_id,nullzero"`
	ReversalBatchID      pulid.ID            `json:"reversalBatchId"      bun:"reversal_batch_id,type:VARCHAR(100),nullzero"                                         keel:"reversal_batch_id,nullzero"`
	ReversedByID         pulid.ID            `json:"reversedById"         bun:"reversed_by_id,type:VARCHAR(100),nullzero"                                            keel:"reversed_by_id,nullzero"`
	ReversedAt           *int64              `json:"reversedAt"           bun:"reversed_at,type:BIGINT,nullzero"                                                     keel:"reversed_at"`
	ReversalReason       string              `json:"reversalReason"       bun:"reversal_reason,type:TEXT,nullzero"                                                   keel:"reversal_reason,nullzero"`
	CreatedByID          pulid.ID            `json:"createdById"          bun:"created_by_id,type:VARCHAR(100),notnull"                                              keel:"created_by_id"`
	UpdatedByID          pulid.ID            `json:"updatedById"          bun:"updated_by_id,type:VARCHAR(100),nullzero"                                             keel:"updated_by_id,nullzero"`
	Version              int64               `json:"version"              bun:"version,type:BIGINT,notnull"                                                          keel:"version,version"`
	CreatedAt            int64               `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Applications []*Application     `json:"applications,omitempty" bun:"rel:has-many,join:id=customer_payment_id"                                                                       keel:"many:customer_payment_id"`
	Customer     *customer.Customer `json:"customer,omitempty"     bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:customer_id"`
}

// Which invoice each dollar of a payment was applied to
type Application struct {
	bun.BaseModel `bun:"table:customer_payment_applications,alias:cpa" json:"-"`
	keel.Table    `                                                             keel:"customer_payment_applications,report=Accounting,label=Payment Application"`

	ID                  pulid.ID `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:cpapp_"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	CustomerPaymentID   pulid.ID `json:"customerPaymentId"   bun:"customer_payment_id,type:VARCHAR(100),notnull"                                        keel:"customer_payment_id"`
	InvoiceID           pulid.ID `json:"invoiceId"           bun:"invoice_id,type:VARCHAR(100),notnull"                                                 keel:"invoice_id"`
	AppliedAmountMinor  int64    `json:"appliedAmountMinor"  bun:"applied_amount_minor,type:BIGINT,notnull"                                             keel:"applied_amount_minor,label=Applied (Minor)"`
	ShortPayAmountMinor int64    `json:"shortPayAmountMinor" bun:"short_pay_amount_minor,type:BIGINT,notnull"                                           keel:"short_pay_amount_minor,label=Short Pay (Minor)"`
	LineNumber          int      `json:"lineNumber"          bun:"line_number,type:INTEGER,notnull"                                                     keel:"line_number,label=Line,dimension"`
	CreatedAt           int64    `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt           int64    `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Payment *Payment         `json:"payment,omitempty" bun:"rel:belongs-to,join:customer_payment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:customer_payment_id,label=Customer Payment"`
	Invoice *invoice.Invoice `json:"invoice,omitempty" bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"          keel:"belongs:invoice_id"`
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

func (p *Payment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("cpay_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

func (p *Payment) BeforeCreate(ctx context.Context) error {
	return p.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (a *Application) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("cpapp_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}

func (a *Application) BeforeCreate(ctx context.Context) error {
	return a.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
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

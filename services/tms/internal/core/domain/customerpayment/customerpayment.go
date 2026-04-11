package customerpayment

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Payment struct {
	bun.BaseModel `bun:"table:customer_payments,alias:cp" json:"-"`

	ID                   pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CustomerID           pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),notnull"`
	PaymentDate          int64    `json:"paymentDate" bun:"payment_date,type:BIGINT,notnull"`
	AccountingDate       int64    `json:"accountingDate" bun:"accounting_date,type:BIGINT,notnull"`
	AmountMinor          int64    `json:"amountMinor" bun:"amount_minor,type:BIGINT,notnull"`
	AppliedAmountMinor   int64    `json:"appliedAmountMinor" bun:"applied_amount_minor,type:BIGINT,notnull,default:0"`
	UnappliedAmountMinor int64    `json:"unappliedAmountMinor" bun:"unapplied_amount_minor,type:BIGINT,notnull,default:0"`
	Status               Status   `json:"status" bun:"status,type:VARCHAR(50),notnull"`
	PaymentMethod        Method   `json:"paymentMethod" bun:"payment_method,type:VARCHAR(50),notnull"`
	ReferenceNumber      string   `json:"referenceNumber" bun:"reference_number,type:VARCHAR(100),nullzero"`
	Memo                 string   `json:"memo" bun:"memo,type:TEXT,nullzero"`
	CurrencyCode         string   `json:"currencyCode" bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	PostedBatchID        pulid.ID `json:"postedBatchId" bun:"posted_batch_id,type:VARCHAR(100),nullzero"`
	ReversalBatchID      pulid.ID `json:"reversalBatchId" bun:"reversal_batch_id,type:VARCHAR(100),nullzero"`
	ReversedByID         pulid.ID `json:"reversedById" bun:"reversed_by_id,type:VARCHAR(100),nullzero"`
	ReversedAt           *int64   `json:"reversedAt" bun:"reversed_at,type:BIGINT,nullzero"`
	ReversalReason       string   `json:"reversalReason" bun:"reversal_reason,type:TEXT,nullzero"`
	CreatedByID          pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID          pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version              int64    `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Applications []*Application `json:"applications,omitempty" bun:"rel:has-many,join:id=customer_payment_id"`
}

type Application struct {
	bun.BaseModel `bun:"table:customer_payment_applications,alias:cpa" json:"-"`

	ID                 pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CustomerPaymentID  pulid.ID `json:"customerPaymentId" bun:"customer_payment_id,type:VARCHAR(100),notnull"`
	InvoiceID          pulid.ID `json:"invoiceId" bun:"invoice_id,type:VARCHAR(100),notnull"`
	AppliedAmountMinor int64    `json:"appliedAmountMinor" bun:"applied_amount_minor,type:BIGINT,notnull"`
	LineNumber         int      `json:"lineNumber" bun:"line_number,type:INTEGER,notnull"`
	CreatedAt          int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *Payment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required),
		validation.Field(&p.BusinessUnitID, validation.Required),
		validation.Field(&p.CustomerID, validation.Required),
		validation.Field(&p.PaymentDate, validation.Required),
		validation.Field(&p.AccountingDate, validation.Required),
		validation.Field(&p.CurrencyCode, validation.Required, validation.Length(3, 3)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if p.AmountMinor <= 0 {
		multiErr.Add("amountMinor", errortypes.ErrInvalid, "Payment amount must be greater than zero")
	}
	if !p.PaymentMethod.IsValid() {
		multiErr.Add("paymentMethod", errortypes.ErrInvalid, "Payment method is invalid")
	}
	for idx, app := range p.Applications {
		if app == nil {
			multiErr.Add("applications", errortypes.ErrInvalid, "Payment applications must not contain null values")
			continue
		}
		app.Validate(multiErr.WithIndex("applications", idx))
	}
}

func (a *Application) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(a,
		validation.Field(&a.InvoiceID, validation.Required),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	if a.AppliedAmountMinor <= 0 {
		multiErr.Add("appliedAmountMinor", errortypes.ErrInvalid, "Applied amount must be greater than zero")
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
	p.UnappliedAmountMinor = p.AmountMinor - applied
	if p.UnappliedAmountMinor < 0 {
		p.UnappliedAmountMinor = 0
	}
	if strings.TrimSpace(p.CurrencyCode) == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}
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

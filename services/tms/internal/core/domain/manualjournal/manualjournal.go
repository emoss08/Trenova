package manualjournal

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Request struct {
	bun.BaseModel `bun:"table:manual_journal_requests,alias:mjr" json:"-"`

	ID                      pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	RequestNumber           string   `json:"requestNumber" bun:"request_number,type:VARCHAR(50),notnull"`
	Status                  Status   `json:"status" bun:"status,type:manual_journal_request_status_enum,notnull,default:'Draft'"`
	Description             string   `json:"description" bun:"description,type:TEXT,notnull"`
	Reason                  string   `json:"reason" bun:"reason,type:TEXT,nullzero"`
	AccountingDate          int64    `json:"accountingDate" bun:"accounting_date,type:BIGINT,notnull"`
	RequestedFiscalYearID   pulid.ID `json:"requestedFiscalYearId" bun:"requested_fiscal_year_id,type:VARCHAR(100),notnull"`
	RequestedFiscalPeriodID pulid.ID `json:"requestedFiscalPeriodId" bun:"requested_fiscal_period_id,type:VARCHAR(100),notnull"`
	CurrencyCode            string   `json:"currencyCode" bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	TotalDebit              int64    `json:"totalDebit" bun:"total_debit_minor,type:BIGINT,notnull,default:0"`
	TotalCredit             int64    `json:"totalCredit" bun:"total_credit_minor,type:BIGINT,notnull,default:0"`
	ApprovedAt              *int64   `json:"approvedAt" bun:"approved_at,type:BIGINT,nullzero"`
	ApprovedByID            pulid.ID `json:"approvedById" bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt              *int64   `json:"rejectedAt" bun:"rejected_at,type:BIGINT,nullzero"`
	RejectedByID            pulid.ID `json:"rejectedById" bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectionReason         string   `json:"rejectionReason" bun:"rejection_reason,type:TEXT,nullzero"`
	CancelledAt             *int64   `json:"cancelledAt" bun:"cancelled_at,type:BIGINT,nullzero"`
	CancelledByID           pulid.ID `json:"cancelledById" bun:"cancelled_by_id,type:VARCHAR(100),nullzero"`
	CancelReason            string   `json:"cancelReason" bun:"cancel_reason,type:TEXT,nullzero"`
	PostedBatchID           pulid.ID `json:"postedBatchId" bun:"posted_batch_id,type:VARCHAR(100),nullzero"`
	CreatedByID             pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID             pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version                 int64    `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt               int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Lines []*Line `json:"lines,omitempty" bun:"rel:has-many,join:id=manual_journal_request_id"`
}

type Line struct {
	bun.BaseModel `bun:"table:manual_journal_request_lines,alias:mjrl" json:"-"`

	ID                     pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ManualJournalRequestID pulid.ID `json:"manualJournalRequestId" bun:"manual_journal_request_id,type:VARCHAR(100),notnull"`
	LineNumber             int      `json:"lineNumber" bun:"line_number,type:INTEGER,notnull"`
	GLAccountID            pulid.ID `json:"glAccountId" bun:"gl_account_id,type:VARCHAR(100),notnull"`
	Description            string   `json:"description" bun:"description,type:TEXT,notnull"`
	DebitAmount            int64    `json:"debitAmount" bun:"debit_amount_minor,type:BIGINT,notnull,default:0"`
	CreditAmount           int64    `json:"creditAmount" bun:"credit_amount_minor,type:BIGINT,notnull,default:0"`
	CustomerID             pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),nullzero"`
	LocationID             pulid.ID `json:"locationId" bun:"location_id,type:VARCHAR(100),nullzero"`
	CreatedAt              int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *Request) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		r,
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(&r.Description, validation.Required),
		validation.Field(&r.AccountingDate, validation.Required),
		validation.Field(&r.CurrencyCode, validation.Required, validation.Length(3, 3)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	for idx, line := range r.Lines {
		if line == nil {
			multiErr.WithIndex("lines", idx).Add("", errortypes.ErrRequired, "Line is required")
			continue
		}

		lineErr := multiErr.WithIndex("lines", idx)
		line.Validate(lineErr)
	}
}

func (l *Line) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		l,
		validation.Field(&l.GLAccountID, validation.Required),
		validation.Field(&l.Description, validation.Required),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if l.DebitAmount < 0 {
		multiErr.Add("debitAmount", errortypes.ErrInvalid, "Debit amount cannot be negative")
	}
	if l.CreditAmount < 0 {
		multiErr.Add("creditAmount", errortypes.ErrInvalid, "Credit amount cannot be negative")
	}
	if (l.DebitAmount == 0 && l.CreditAmount == 0) || (l.DebitAmount > 0 && l.CreditAmount > 0) {
		multiErr.Add("debitAmount", errortypes.ErrInvalid, "Exactly one of debit or credit amount must be greater than zero")
	}
}

func (r *Request) SyncTotals() {
	var totalDebit int64
	var totalCredit int64

	for idx, line := range r.Lines {
		if line == nil {
			continue
		}

		line.LineNumber = idx + 1
		totalDebit += line.DebitAmount
		totalCredit += line.CreditAmount
	}

	r.TotalDebit = totalDebit
	r.TotalCredit = totalCredit
}

func (r *Request) IsBalanced() bool {
	return r.TotalDebit > 0 && r.TotalDebit == r.TotalCredit
}

func (r *Request) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("mjr_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (l *Line) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("mjrl_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}

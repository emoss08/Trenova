package manualjournal

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ domaintypes.PostgresSearchable = (*Request)(nil)

type Request struct {
	bun.BaseModel             `bun:"table:manual_journal_requests,alias:mjr" json:"-"`
	keel.Table                `keel:"manual_journal_requests"`
	pagination.CursorValueSet `bun:",embed"                                  json:"-"`

	ID                      pulid.ID `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:mjr_"`
	OrganizationID          pulid.ID `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	RequestNumber           string   `json:"requestNumber"           bun:"request_number,type:VARCHAR(50),notnull" keel:"request_number"`
	Status                  Status   `json:"status"                  bun:"status,type:manual_journal_request_status_enum,notnull,default:'Draft'" keel:"status"`
	Description             string   `json:"description"             bun:"description,type:TEXT,notnull" keel:"description"`
	Reason                  string   `json:"reason"                  bun:"reason,type:TEXT,nullzero" keel:"reason,nullzero"`
	AccountingDate          int64    `json:"accountingDate"          bun:"accounting_date,type:BIGINT,notnull" keel:"accounting_date"`
	RequestedFiscalYearID   pulid.ID `json:"requestedFiscalYearId"   bun:"requested_fiscal_year_id,type:VARCHAR(100),notnull" keel:"requested_fiscal_year_id"`
	RequestedFiscalPeriodID pulid.ID `json:"requestedFiscalPeriodId" bun:"requested_fiscal_period_id,type:VARCHAR(100),notnull" keel:"requested_fiscal_period_id"`
	CurrencyCode            string   `json:"currencyCode"            bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	TotalDebit              int64    `json:"totalDebit"              bun:"total_debit_minor,type:BIGINT,notnull" keel:"total_debit_minor"`
	TotalCredit             int64    `json:"totalCredit"             bun:"total_credit_minor,type:BIGINT,notnull" keel:"total_credit_minor"`
	ApprovedAt              *int64   `json:"approvedAt"              bun:"approved_at,type:BIGINT,nullzero" keel:"approved_at"`
	ApprovedByID            pulid.ID `json:"approvedById"            bun:"approved_by_id,type:VARCHAR(100),nullzero" keel:"approved_by_id,nullzero"`
	RejectedAt              *int64   `json:"rejectedAt"              bun:"rejected_at,type:BIGINT,nullzero" keel:"rejected_at"`
	RejectedByID            pulid.ID `json:"rejectedById"            bun:"rejected_by_id,type:VARCHAR(100),nullzero" keel:"rejected_by_id,nullzero"`
	RejectionReason         string   `json:"rejectionReason"         bun:"rejection_reason,type:TEXT,nullzero" keel:"rejection_reason,nullzero"`
	CancelledAt             *int64   `json:"cancelledAt"             bun:"cancelled_at,type:BIGINT,nullzero" keel:"cancelled_at"`
	CancelledByID           pulid.ID `json:"cancelledById"           bun:"cancelled_by_id,type:VARCHAR(100),nullzero" keel:"cancelled_by_id,nullzero"`
	CancelReason            string   `json:"cancelReason"            bun:"cancel_reason,type:TEXT,nullzero" keel:"cancel_reason,nullzero"`
	PostedBatchID           pulid.ID `json:"postedBatchId"           bun:"posted_batch_id,type:VARCHAR(100),nullzero" keel:"posted_batch_id,nullzero"`
	CreatedByID             pulid.ID `json:"createdById"             bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	UpdatedByID             pulid.ID `json:"updatedById"             bun:"updated_by_id,type:VARCHAR(100),nullzero" keel:"updated_by_id,nullzero"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Lines []*Line `json:"lines,omitempty" bun:"rel:has-many,join:id=manual_journal_request_id" keel:"many:manual_journal_request_id"`
}

type Line struct {
	bun.BaseModel             `bun:"table:manual_journal_request_lines,alias:mjrl" json:"-"`
	keel.Table                `keel:"manual_journal_request_lines"`
	pagination.CursorValueSet `bun:",embed"                                        json:"-"`

	ID                     pulid.ID `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:mjrl_"`
	OrganizationID         pulid.ID `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	ManualJournalRequestID pulid.ID `json:"manualJournalRequestId" bun:"manual_journal_request_id,type:VARCHAR(100),notnull" keel:"manual_journal_request_id"`
	LineNumber             int      `json:"lineNumber"             bun:"line_number,type:INTEGER,notnull" keel:"line_number"`
	GLAccountID            pulid.ID `json:"glAccountId"            bun:"gl_account_id,type:VARCHAR(100),notnull" keel:"gl_account_id"`
	Description            string   `json:"description"            bun:"description,type:TEXT,notnull" keel:"description"`
	DebitAmount            int64    `json:"debitAmount"            bun:"debit_amount_minor,type:BIGINT,notnull" keel:"debit_amount_minor"`
	CreditAmount           int64    `json:"creditAmount"           bun:"credit_amount_minor,type:BIGINT,notnull" keel:"credit_amount_minor"`
	CustomerID             pulid.ID `json:"customerId"             bun:"customer_id,type:VARCHAR(100),nullzero" keel:"customer_id,nullzero"`
	LocationID             pulid.ID `json:"locationId"             bun:"location_id,type:VARCHAR(100),nullzero" keel:"location_id,nullzero"`
	CreatedAt              int64    `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt              int64    `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	GLAccount *glaccount.GLAccount `json:"glAccount,omitempty" bun:"rel:belongs-to,join:gl_account_id=id" keel:"belongs:gl_account_id"`
}

func (r *Request) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(&r.Description, validation.Required),
		validation.Field(&r.AccountingDate, validation.Required),
		validation.Field(&r.CurrencyCode, validation.Required, validation.Length(3, 3)),
	))

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
	multiErr.AddOzzoError(validation.ValidateStruct(
		l,
		validation.Field(&l.GLAccountID, validation.Required),
		validation.Field(&l.Description, validation.Required),
	))

	if l.DebitAmount < 0 {
		multiErr.Add("debitAmount", errortypes.ErrInvalid, "Debit amount cannot be negative")
	}
	if l.CreditAmount < 0 {
		multiErr.Add("creditAmount", errortypes.ErrInvalid, "Credit amount cannot be negative")
	}
	if (l.DebitAmount == 0 && l.CreditAmount == 0) || (l.DebitAmount > 0 && l.CreditAmount > 0) {
		multiErr.Add(
			"debitAmount",
			errortypes.ErrInvalid,
			"Exactly one of debit or credit amount must be greater than zero",
		)
	}
}

func (r *Request) GetID() pulid.ID {
	return r.ID
}

func (r *Request) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *Request) GetTableName() string {
	return "manual_journal_requests"
}

func (r *Request) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "mjr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "request_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "reason", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
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

func (r *Request) BeforeCreate(ctx context.Context) error {
	return r.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
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

func (l *Line) BeforeCreate(ctx context.Context) error {
	return l.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

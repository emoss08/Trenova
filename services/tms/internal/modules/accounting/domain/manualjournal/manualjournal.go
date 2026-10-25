package manualjournal

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var _ domaintypes.PostgresSearchable = (*Request)(nil)

type Request struct {
	keel.Table `keel:"manual_journal_requests"`

	ID                      pulid.ID `json:"id"                      keel:"id,pk,autoid:mjr_"`
	OrganizationID          pulid.ID `json:"organizationId"          keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	RequestNumber           string   `json:"requestNumber"           keel:"request_number"`
	Status                  Status   `json:"status"                  keel:"status"`
	Description             string   `json:"description"             keel:"description"`
	Reason                  string   `json:"reason"                  keel:"reason,nullzero"`
	AccountingDate          int64    `json:"accountingDate"          keel:"accounting_date"`
	RequestedFiscalYearID   pulid.ID `json:"requestedFiscalYearId"   keel:"requested_fiscal_year_id"`
	RequestedFiscalPeriodID pulid.ID `json:"requestedFiscalPeriodId" keel:"requested_fiscal_period_id"`
	CurrencyCode            string   `json:"currencyCode"            keel:"currency_code"`
	TotalDebit              int64    `json:"totalDebit"              keel:"total_debit_minor"`
	TotalCredit             int64    `json:"totalCredit"             keel:"total_credit_minor"`
	ApprovedAt              *int64   `json:"approvedAt"              keel:"approved_at"`
	ApprovedByID            pulid.ID `json:"approvedById"            keel:"approved_by_id,nullzero"`
	RejectedAt              *int64   `json:"rejectedAt"              keel:"rejected_at"`
	RejectedByID            pulid.ID `json:"rejectedById"            keel:"rejected_by_id,nullzero"`
	RejectionReason         string   `json:"rejectionReason"         keel:"rejection_reason,nullzero"`
	CancelledAt             *int64   `json:"cancelledAt"             keel:"cancelled_at"`
	CancelledByID           pulid.ID `json:"cancelledById"           keel:"cancelled_by_id,nullzero"`
	CancelReason            string   `json:"cancelReason"            keel:"cancel_reason,nullzero"`
	PostedBatchID           pulid.ID `json:"postedBatchId"           keel:"posted_batch_id,nullzero"`
	CreatedByID             pulid.ID `json:"createdById"             keel:"created_by_id"`
	UpdatedByID             pulid.ID `json:"updatedById"             keel:"updated_by_id,nullzero"`

	SearchVector string `json:"-"`
	Rank         string `json:"-"         keel:"rank,computed"`
	Version      int64  `json:"version"   keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Lines []*Line `json:"lines,omitempty" keel:"many:manual_journal_request_id"`
}

type Line struct {
	keel.Table `keel:"manual_journal_request_lines"`

	ID                     pulid.ID `json:"id"                     keel:"id,pk,autoid:mjrl_"`
	OrganizationID         pulid.ID `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	ManualJournalRequestID pulid.ID `json:"manualJournalRequestId" keel:"manual_journal_request_id"`
	LineNumber             int      `json:"lineNumber"             keel:"line_number"`
	GLAccountID            pulid.ID `json:"glAccountId"            keel:"gl_account_id"`
	Description            string   `json:"description"            keel:"description"`
	DebitAmount            int64    `json:"debitAmount"            keel:"debit_amount_minor"`
	CreditAmount           int64    `json:"creditAmount"           keel:"credit_amount_minor"`
	CustomerID             pulid.ID `json:"customerId"             keel:"customer_id,nullzero"`
	LocationID             pulid.ID `json:"locationId"             keel:"location_id,nullzero"`
	CreatedAt              int64    `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64    `json:"updatedAt"              keel:"updated_at,autoupdate"`

	GLAccount *glaccount.GLAccount `json:"glAccount,omitempty" keel:"belongs:gl_account_id"`
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

func (r *Request) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = StatusDraft
	}
	if r.CurrencyCode == "" {
		r.CurrencyCode = "USD"
	}

	return nil
}

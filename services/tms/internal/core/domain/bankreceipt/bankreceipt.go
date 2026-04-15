package bankreceipt

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Receipt struct {
	bun.BaseModel `bun:"table:bank_receipts,alias:br" json:"-"`

	ID                       pulid.ID `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID           pulid.ID `json:"organizationId"           bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID `json:"businessUnitId"           bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ReceiptDate              int64    `json:"receiptDate"              bun:"receipt_date,type:BIGINT,notnull"`
	AmountMinor              int64    `json:"amountMinor"              bun:"amount_minor,type:BIGINT,notnull"`
	ReferenceNumber          string   `json:"referenceNumber"          bun:"reference_number,type:VARCHAR(100),nullzero"`
	Memo                     string   `json:"memo"                     bun:"memo,type:TEXT,nullzero"`
	Status                   Status   `json:"status"                   bun:"status,type:VARCHAR(50),notnull"`
	ImportBatchID            pulid.ID `json:"importBatchId"            bun:"import_batch_id,type:VARCHAR(100),nullzero"`
	MatchedCustomerPaymentID pulid.ID `json:"matchedCustomerPaymentId" bun:"matched_customer_payment_id,type:VARCHAR(100),nullzero"`
	MatchedAt                *int64   `json:"matchedAt"                bun:"matched_at,type:BIGINT,nullzero"`
	MatchedByID              pulid.ID `json:"matchedById"              bun:"matched_by_id,type:VARCHAR(100),nullzero"`
	ExceptionReason          string   `json:"exceptionReason"          bun:"exception_reason,type:TEXT,nullzero"`
	CreatedByID              pulid.ID `json:"createdById"              bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID              pulid.ID `json:"updatedById"              bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version                  int64    `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64    `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64    `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type MatchSuggestion struct {
	CustomerPaymentID pulid.ID `json:"customerPaymentId"`
	ReferenceNumber   string   `json:"referenceNumber"`
	AmountMinor       int64    `json:"amountMinor"`
	CustomerID        pulid.ID `json:"customerId"`
	Score             int      `json:"score"`
	Reason            string   `json:"reason"`
}

type ExceptionAging struct {
	CurrentCount    int64 `json:"currentCount"`
	CurrentAmount   int64 `json:"currentAmount"`
	Days1To3Count   int64 `json:"days1To3Count"`
	Days1To3Amount  int64 `json:"days1To3Amount"`
	Days4To7Count   int64 `json:"days4To7Count"`
	Days4To7Amount  int64 `json:"days4To7Amount"`
	DaysOver7Count  int64 `json:"daysOver7Count"`
	DaysOver7Amount int64 `json:"daysOver7Amount"`
}

type ReconciliationSummary struct {
	AsOfDate              int64          `json:"asOfDate"`
	ImportedCount         int64          `json:"importedCount"`
	ImportedAmount        int64          `json:"importedAmount"`
	MatchedCount          int64          `json:"matchedCount"`
	MatchedAmount         int64          `json:"matchedAmount"`
	ExceptionCount        int64          `json:"exceptionCount"`
	ExceptionAmount       int64          `json:"exceptionAmount"`
	ActiveWorkItemCount   int64          `json:"activeWorkItemCount"`
	AssignedWorkItemCount int64          `json:"assignedWorkItemCount"`
	InReviewWorkItemCount int64          `json:"inReviewWorkItemCount"`
	ExceptionAging        ExceptionAging `json:"exceptionAging"`
}

func (r *Receipt) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(&r.ReceiptDate, validation.Required),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
	if r.AmountMinor <= 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Receipt amount must be greater than zero",
		)
	}
}

func (r *Receipt) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("brcpt_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}

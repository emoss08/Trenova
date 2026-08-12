package journalreversal

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ domaintypes.PostgresSearchable = (*Reversal)(nil)
	_ pagination.CursorEntity        = (*Reversal)(nil)
)

type Reversal struct {
	bun.BaseModel `bun:"table:journal_reversals,alias:jr" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                      pulid.ID `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OriginalJournalEntryID  pulid.ID `json:"originalJournalEntryId"  bun:"original_journal_entry_id,type:VARCHAR(100),notnull"`
	ReversalJournalEntryID  pulid.ID `json:"reversalJournalEntryId"  bun:"reversal_journal_entry_id,type:VARCHAR(100),nullzero"`
	PostedBatchID           pulid.ID `json:"postedBatchId"           bun:"posted_batch_id,type:VARCHAR(100),nullzero"`
	Status                  Status   `json:"status"                  bun:"status,type:journal_reversal_status_enum,notnull"`
	RequestedAccountingDate int64    `json:"requestedAccountingDate" bun:"requested_accounting_date,type:BIGINT,notnull"`
	ResolvedFiscalYearID    pulid.ID `json:"resolvedFiscalYearId"    bun:"resolved_fiscal_year_id,type:VARCHAR(100),notnull"`
	ResolvedFiscalPeriodID  pulid.ID `json:"resolvedFiscalPeriodId"  bun:"resolved_fiscal_period_id,type:VARCHAR(100),notnull"`
	ReasonCode              string   `json:"reasonCode"              bun:"reason_code,type:VARCHAR(100),notnull"`
	ReasonText              string   `json:"reasonText"              bun:"reason_text,type:TEXT,notnull"`
	RequestedByID           pulid.ID `json:"requestedById"           bun:"requested_by_id,type:VARCHAR(100),notnull"`
	ApprovedByID            pulid.ID `json:"approvedById"            bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt              *int64   `json:"approvedAt"              bun:"approved_at,type:BIGINT,nullzero"`
	RejectedByID            pulid.ID `json:"rejectedById"            bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt              *int64   `json:"rejectedAt"              bun:"rejected_at,type:BIGINT,nullzero"`
	RejectionReason         string   `json:"rejectionReason"         bun:"rejection_reason,type:TEXT,nullzero"`
	CancelledByID           pulid.ID `json:"cancelledById"           bun:"cancelled_by_id,type:VARCHAR(100),nullzero"`
	CancelledAt             *int64   `json:"cancelledAt"             bun:"cancelled_at,type:BIGINT,nullzero"`
	CancelReason            string   `json:"cancelReason"            bun:"cancel_reason,type:TEXT,nullzero"`
	PostedByID              pulid.ID `json:"postedById"              bun:"posted_by_id,type:VARCHAR(100),nullzero"`
	PostedAt                *int64   `json:"postedAt"                bun:"posted_at,type:BIGINT,nullzero"`
	CreatedAt               int64    `json:"createdAt"               bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt               int64    `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull"`
	Version                 int64    `json:"version"                 bun:"version,type:BIGINT,notnull"`
}

func (r *Reversal) GetID() pulid.ID {
	return r.ID
}

func (r *Reversal) GetCreatedAt() int64 {
	return r.CreatedAt
}

func (r *Reversal) GetTableName() string {
	return "journal_reversals"
}

func (r *Reversal) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "jr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "reason_code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "reason_text",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (r *Reversal) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.OriginalJournalEntryID,
			validation.Required.Error("Original journal entry is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		// The accounting date decides which period the reversal lands in, so an
		// unset one would post the entry into an undefined period.
		validation.Field(&r.RequestedAccountingDate,
			validation.Required.Error("Requested accounting date is required"),
			validation.Min(int64(1)).Error("Requested accounting date must be a valid date"),
		),
		validation.Field(&r.ResolvedFiscalYearID,
			validation.Required.Error("Resolved fiscal year is required"),
		),
		validation.Field(&r.ResolvedFiscalPeriodID,
			validation.Required.Error("Resolved fiscal period is required"),
		),
		// Reversing a posted entry rewrites the books, so the reason is part of
		// the record rather than something an auditor has to reconstruct.
		validation.Field(&r.ReasonCode,
			validation.Required.Error("Reason code is required"),
			validation.Length(1, maxReasonCodeLength).
				Error("Reason code cannot be longer than 100 characters"),
		),
		validation.Field(&r.ReasonText, validation.Required.Error("Reason is required")),
		validation.Field(&r.RequestedByID, validation.Required.Error("Requested by is required")),
		validation.Field(&r.ApprovedAt,
			validation.When(r.ApprovedByID.IsNotNil(), validation.Required.Error(
				"An approved reversal must record when it was approved",
			)),
		),
		validation.Field(&r.RejectedAt,
			validation.When(r.RejectedByID.IsNotNil(), validation.Required.Error(
				"A rejected reversal must record when it was rejected",
			)),
		),
		validation.Field(&r.RejectionReason,
			validation.When(
				r.Status == StatusRejected,
				validation.Required.Error("A rejected reversal must record why"),
			),
		),
		validation.Field(&r.CancelledAt,
			validation.When(r.CancelledByID.IsNotNil(), validation.Required.Error(
				"A cancelled reversal must record when it was cancelled",
			)),
		),
		validation.Field(&r.PostedAt,
			validation.When(r.PostedByID.IsNotNil(), validation.Required.Error(
				"A posted reversal must record when it was posted",
			)),
		),
		validation.Field(&r.ReversalJournalEntryID,
			validation.When(
				r.Status == StatusPosted,
				validation.Required.Error("A posted reversal must name the entry it created"),
			),
		),
	))
}

const maxReasonCodeLength = 100

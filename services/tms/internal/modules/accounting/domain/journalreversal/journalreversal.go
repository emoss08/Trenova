package journalreversal

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// IDPrefix is the prefix every reversal's id carries. The service and the
// repository mint the id rather than an insert hook, so the prefix is named
// here for both of them and for the table that tells ids apart by prefix.
const IDPrefix = "jrev_"

var (
	_ domaintypes.PostgresSearchable = (*Reversal)(nil)
	_ pagination.CursorEntity        = (*Reversal)(nil)
)

type Reversal struct {
	keel.Table `keel:"journal_reversals"`

	ID                      pulid.ID `json:"id"                      keel:"id,pk"`
	OrganizationID          pulid.ID `json:"organizationId"          keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OriginalJournalEntryID  pulid.ID `json:"originalJournalEntryId"  keel:"original_journal_entry_id"`
	ReversalJournalEntryID  pulid.ID `json:"reversalJournalEntryId"  keel:"reversal_journal_entry_id,nullzero"`
	PostedBatchID           pulid.ID `json:"postedBatchId"           keel:"posted_batch_id,nullzero"`
	Status                  Status   `json:"status"                  keel:"status"`
	RequestedAccountingDate int64    `json:"requestedAccountingDate" keel:"requested_accounting_date"`
	ResolvedFiscalYearID    pulid.ID `json:"resolvedFiscalYearId"    keel:"resolved_fiscal_year_id"`
	ResolvedFiscalPeriodID  pulid.ID `json:"resolvedFiscalPeriodId"  keel:"resolved_fiscal_period_id"`
	ReasonCode              string   `json:"reasonCode"              keel:"reason_code"`
	ReasonText              string   `json:"reasonText"              keel:"reason_text"`
	RequestedByID           pulid.ID `json:"requestedById"           keel:"requested_by_id"`
	ApprovedByID            pulid.ID `json:"approvedById"            keel:"approved_by_id,nullzero"`
	ApprovedAt              *int64   `json:"approvedAt"              keel:"approved_at"`
	RejectedByID            pulid.ID `json:"rejectedById"            keel:"rejected_by_id,nullzero"`
	RejectedAt              *int64   `json:"rejectedAt"              keel:"rejected_at"`
	RejectionReason         string   `json:"rejectionReason"         keel:"rejection_reason,nullzero"`
	CancelledByID           pulid.ID `json:"cancelledById"           keel:"cancelled_by_id,nullzero"`
	CancelledAt             *int64   `json:"cancelledAt"             keel:"cancelled_at"`
	CancelReason            string   `json:"cancelReason"            keel:"cancel_reason,nullzero"`
	PostedByID              pulid.ID `json:"postedById"              keel:"posted_by_id,nullzero"`
	PostedAt                *int64   `json:"postedAt"                keel:"posted_at"`
	CreatedAt               int64    `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64    `json:"updatedAt"               keel:"updated_at,autoupdate"`
	Version                 int64    `json:"version"                 keel:"version,version"`

	OriginalJournalEntry *journalentry.JournalEntry `json:"originalJournalEntry,omitempty" keel:"belongs:original_journal_entry_id"`
	ReversalJournalEntry *journalentry.JournalEntry `json:"reversalJournalEntry,omitempty" keel:"belongs:reversal_journal_entry_id"`
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
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "originalJournalEntry", Queryable: true},
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

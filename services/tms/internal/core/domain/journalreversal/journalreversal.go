package journalreversal

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	Version                 int64    `json:"version"                 bun:"version,type:BIGINT,notnull,default:0"`
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
			{Name: "reason_code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "reason_text", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

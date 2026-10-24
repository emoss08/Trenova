package journalentry

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// The unit journal entries are posted in
type JournalBatch struct {
	keel.Table `                                              keel:"journal_batches,report=Accounting"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:jb_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BatchNumber    string   `json:"batchNumber" keel:"batch_number"`
	BatchType      string   `json:"batchType" keel:"batch_type"`
	Status         string   `json:"status" keel:"status"`
	Description    string   `json:"description" keel:"description"`
	AccountingDate int64    `json:"accountingDate" keel:"accounting_date"`
	FiscalYearID   pulid.ID `json:"fiscalYearId" keel:"fiscal_year_id"`
	FiscalPeriodID pulid.ID `json:"fiscalPeriodId" keel:"fiscal_period_id"`
	EntryCount     int      `json:"entryCount" keel:"entry_count,label=Entries,format=count"`
	PostedAt       *int64   `json:"postedAt" keel:"posted_at"`
	PostedByID     pulid.ID `json:"postedById" keel:"posted_by_id,nullzero"`
	CreatedByID    pulid.ID `json:"createdById" keel:"created_by_id"`
	UpdatedByID    pulid.ID `json:"updatedById" keel:"updated_by_id,nullzero"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	FiscalYear   *fiscalyear.FiscalYear     `json:"fiscalYear,omitempty" keel:"belongs:fiscal_year_id"`
	FiscalPeriod *fiscalperiod.FiscalPeriod `json:"fiscalPeriod,omitempty" keel:"belongs:fiscal_period_id"`
	Entries      []*JournalEntry            `json:"entries,omitempty" keel:"many:batch_id,label=Journal Entries"`
}

func (b *JournalBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.BatchNumber, validation.Required, validation.Length(1, 50)),
		validation.Field(&b.BatchType, validation.Required, validation.Length(1, 50)),
		validation.Field(&b.Status, validation.Required, validation.Length(1, 50)),
		validation.Field(&b.Description, validation.Required),
		validation.Field(&b.FiscalYearID, validation.Required),
		validation.Field(&b.FiscalPeriodID, validation.Required),
		validation.Field(&b.CreatedByID, validation.Required),
	))

	if b.EntryCount < 0 {
		multiErr.Add("entryCount", errortypes.ErrInvalid, "Entry count cannot be negative")
	}
}

func (b *JournalBatch) GetTableName() string { return "journal_batches" }

func (b *JournalBatch) GetID() pulid.ID { return b.ID }

func (b *JournalBatch) GetCreatedAt() int64 { return b.CreatedAt }

func (b *JournalBatch) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *JournalBatch) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

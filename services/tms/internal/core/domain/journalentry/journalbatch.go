package journalentry

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*JournalBatch)(nil)

// BatchType and Status stay plain strings. Their value sets are produced by the
// posting workflows rather than declared in one place, and the column is a
// varchar rather than a database enum, so a Go enum here would describe a
// narrower set than the table actually holds.
type JournalBatch struct {
	bun.BaseModel `bun:"table:journal_batches,alias:jb" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BatchNumber    string   `json:"batchNumber"    bun:"batch_number,type:VARCHAR(50),notnull"`
	BatchType      string   `json:"batchType"      bun:"batch_type,type:VARCHAR(50),notnull"`
	Status         string   `json:"status"         bun:"status,type:VARCHAR(50),notnull"`
	Description    string   `json:"description"    bun:"description,type:TEXT,notnull"`
	AccountingDate int64    `json:"accountingDate" bun:"accounting_date,type:BIGINT,notnull"`
	FiscalYearID   pulid.ID `json:"fiscalYearId"   bun:"fiscal_year_id,type:VARCHAR(100),notnull"`
	FiscalPeriodID pulid.ID `json:"fiscalPeriodId" bun:"fiscal_period_id,type:VARCHAR(100),notnull"`
	EntryCount     int      `json:"entryCount"     bun:"entry_count,type:INTEGER,notnull"`
	PostedAt       *int64   `json:"postedAt"       bun:"posted_at,type:BIGINT,nullzero"`
	PostedByID     pulid.ID `json:"postedById"     bun:"posted_by_id,type:VARCHAR(100),nullzero"`
	CreatedByID    pulid.ID `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),notnull"`
	UpdatedByID    pulid.ID `json:"updatedById"    bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	FiscalYear   *fiscalyear.FiscalYear     `json:"fiscalYear,omitempty"   bun:"rel:belongs-to,join:fiscal_year_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FiscalPeriod *fiscalperiod.FiscalPeriod `json:"fiscalPeriod,omitempty" bun:"rel:belongs-to,join:fiscal_period_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Entries      []*JournalEntry            `json:"entries,omitempty"      bun:"rel:has-many,join:id=batch_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (b *JournalBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("jb_")
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}
	return nil
}

package glbalance

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*GLAccountPeriodBalance)(nil)

// GLAccountPeriodBalance is the trial balance, one row per account per period.
// It has no id column: its identity is the five-column key below, which is why
// it implements neither GetID nor GetCreatedAt and is not cursor-paginated.
type GLAccountPeriodBalance struct {
	bun.BaseModel `bun:"table:gl_account_balances_by_period,alias:gb" json:"-"`

	OrganizationID     pulid.ID `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	GLAccountID        pulid.ID `json:"glAccountId"        bun:"gl_account_id,pk,type:VARCHAR(100),notnull"`
	FiscalYearID       pulid.ID `json:"fiscalYearId"       bun:"fiscal_year_id,pk,type:VARCHAR(100),notnull"`
	FiscalPeriodID     pulid.ID `json:"fiscalPeriodId"     bun:"fiscal_period_id,pk,type:VARCHAR(100),notnull"`
	PeriodDebitMinor   int64    `json:"periodDebitMinor"   bun:"period_debit_minor,type:BIGINT,notnull"`
	PeriodCreditMinor  int64    `json:"periodCreditMinor"  bun:"period_credit_minor,type:BIGINT,notnull"`
	NetChangeMinor     int64    `json:"netChangeMinor"     bun:"net_change_minor,type:BIGINT,notnull"`
	LastJournalEntryID pulid.ID `json:"lastJournalEntryId" bun:"last_journal_entry_id,type:VARCHAR(100),nullzero"`
	CreatedAt          int64    `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64    `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	GLAccount        *glaccount.GLAccount       `json:"glAccount,omitempty"        bun:"rel:belongs-to,join:gl_account_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FiscalYear       *fiscalyear.FiscalYear     `json:"fiscalYear,omitempty"       bun:"rel:belongs-to,join:fiscal_year_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	FiscalPeriod     *fiscalperiod.FiscalPeriod `json:"fiscalPeriod,omitempty"     bun:"rel:belongs-to,join:fiscal_period_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	LastJournalEntry *journalentry.JournalEntry `json:"lastJournalEntry,omitempty" bun:"rel:belongs-to,join:last_journal_entry_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (b *GLAccountPeriodBalance) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(b,
		validation.Field(&b.OrganizationID, validation.Required),
		validation.Field(&b.BusinessUnitID, validation.Required),
		validation.Field(&b.GLAccountID, validation.Required),
		validation.Field(&b.FiscalYearID, validation.Required),
		validation.Field(&b.FiscalPeriodID, validation.Required),
	))

	if b.PeriodDebitMinor < 0 || b.PeriodCreditMinor < 0 {
		multiErr.Add(
			"periodDebitMinor",
			errortypes.ErrInvalid,
			"Period debit and credit cannot be negative",
		)
	}
	if b.NetChangeMinor != b.PeriodDebitMinor-b.PeriodCreditMinor {
		multiErr.Add(
			"netChangeMinor",
			errortypes.ErrInvalid,
			"Net change must equal period debit less period credit",
		)
	}
}

func (b *GLAccountPeriodBalance) GetTableName() string {
	return "gl_account_balances_by_period"
}

func (b *GLAccountPeriodBalance) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *GLAccountPeriodBalance) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *GLAccountPeriodBalance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}
	return nil
}

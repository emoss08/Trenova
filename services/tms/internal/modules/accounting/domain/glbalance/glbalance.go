package glbalance

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Debits, credits and net change per GL account per fiscal period
type GLAccountPeriodBalance struct {
	keel.Table `keel:"gl_account_balances_by_period,report=Accounting,label=Account Period Balance"`

	OrganizationID     pulid.ID `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	GLAccountID        pulid.ID `json:"glAccountId"        keel:"gl_account_id,pk"`
	FiscalYearID       pulid.ID `json:"fiscalYearId"       keel:"fiscal_year_id,pk"`
	FiscalPeriodID     pulid.ID `json:"fiscalPeriodId"     keel:"fiscal_period_id,pk"`
	PeriodDebitMinor   int64    `json:"periodDebitMinor"   keel:"period_debit_minor,label=Period Debit (Minor)"`
	PeriodCreditMinor  int64    `json:"periodCreditMinor"  keel:"period_credit_minor,label=Period Credit (Minor)"`
	NetChangeMinor     int64    `json:"netChangeMinor"     keel:"net_change_minor,label=Net Change (Minor)"`
	LastJournalEntryID pulid.ID `json:"lastJournalEntryId" keel:"last_journal_entry_id,nullzero"`
	CreatedAt          int64    `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64    `json:"updatedAt"          keel:"updated_at,autoupdate"`

	GLAccount        *glaccount.GLAccount       `json:"glAccount,omitempty"        keel:"belongs:gl_account_id"`
	FiscalYear       *fiscalyear.FiscalYear     `json:"fiscalYear,omitempty"       keel:"belongs:fiscal_year_id"`
	FiscalPeriod     *fiscalperiod.FiscalPeriod `json:"fiscalPeriod,omitempty"     keel:"belongs:fiscal_period_id"`
	LastJournalEntry *journalentry.JournalEntry `json:"lastJournalEntry,omitempty" keel:"belongs:last_journal_entry_id"`
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

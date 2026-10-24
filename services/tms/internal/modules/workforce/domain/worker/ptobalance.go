package worker

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*WorkerPTOBalance)(nil)
)

// What each worker has accrued, used and carried, per time-off type
type WorkerPTOBalance struct {
	keel.Table `                                                   keel:"worker_pto_balances,report=Parties,label=PTO Balance"`

	ID                   pulid.ID        `json:"id" keel:"id,pk,autoid:wpb_"`
	BusinessUnitID       pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID             pulid.ID        `json:"workerId" keel:"worker_id"`
	PTOType              PTOType         `json:"ptoType" keel:"pto_type,label=Time Off Type"`
	BalanceDays          decimal.Decimal `json:"balanceDays" keel:"balance_days,label=Balance (days)"`
	AccruedYTDDays       decimal.Decimal `json:"accruedYtdDays" keel:"accrued_ytd_days,label=Accrued YTD (days)"`
	UsedYTDDays          decimal.Decimal `json:"usedYtdDays" keel:"used_ytd_days,label=Used YTD (days)"`
	CarriedDays          decimal.Decimal `json:"carriedDays" keel:"carried_days,label=Carried (days)"`
	EntryCount           int64           `json:"entryCount" keel:"entry_count"`
	LastAccrualPeriodKey string          `json:"lastAccrualPeriodKey" keel:"last_accrual_period_key,nullzero"`
	LastRolloverKey      string          `json:"lastRolloverKey" keel:"last_rollover_key,nullzero"`
	YearStartedAt        *int64          `json:"yearStartedAt" keel:"year_started_at"`
	Version              int64           `json:"version" keel:"version,version"`
	CreatedAt            int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64           `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker *Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (b *WorkerPTOBalance) Validate(multiErr *errortypes.MultiError) {
	if b.WorkerID.IsNil() {
		multiErr.Add("workerId", errortypes.ErrRequired, "Worker is required")
	}
	if err := domainvalidation.ValidEnum[PTOType](
		"ptoType must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
	).Validate(b.PTOType); err != nil {
		multiErr.Add("ptoType", errortypes.ErrInvalid, err.Error())
	}
}

func (b *WorkerPTOBalance) Apply(entry *WorkerPTOLedgerEntry) {
	b.EntryCount++
	entry.Sequence = b.EntryCount
	b.BalanceDays = b.BalanceDays.Add(entry.AmountDays)
	entry.BalanceAfterDays = b.BalanceDays

	switch entry.EntryType {
	case PTOLedgerEntryAccrual, PTOLedgerEntryOpeningBalance, PTOLedgerEntryCarryover:
		b.AccruedYTDDays = b.AccruedYTDDays.Add(entry.AmountDays)
	case PTOLedgerEntryUsage:
		b.UsedYTDDays = b.UsedYTDDays.Add(entry.AmountDays.Neg())
	case PTOLedgerEntryReversal:
		b.UsedYTDDays = b.UsedYTDDays.Sub(entry.AmountDays)
	case PTOLedgerEntryAdjustment, PTOLedgerEntryExpiry, PTOLedgerEntryPayout,
		PTOLedgerEntryForfeiture:
	}
}

func (b *WorkerPTOBalance) GetID() pulid.ID { return b.ID }

func (b *WorkerPTOBalance) GetCreatedAt() int64 { return b.CreatedAt }

func (b *WorkerPTOBalance) GetOrganizationID() pulid.ID { return b.OrganizationID }

func (b *WorkerPTOBalance) GetBusinessUnitID() pulid.ID { return b.BusinessUnitID }

func (b *WorkerPTOBalance) GetTableName() string { return "worker_pto_balances" }

func (b *WorkerPTOBalance) GetResourceType() string { return "worker_pto_balance" }

func (b *WorkerPTOBalance) GetResourceID() string { return b.ID.String() }

package worker

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPTOBalance)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTOBalance)(nil)
)

type WorkerPTOBalance struct {
	bun.BaseModel `bun:"table:worker_pto_balances,alias:wpb" json:"-"`

	ID                   pulid.ID        `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID        `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID        `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID             pulid.ID        `json:"workerId"             bun:"worker_id,type:VARCHAR(100),notnull"`
	PTOType              PTOType         `json:"ptoType"              bun:"pto_type,type:worker_pto_type_enum,notnull"`
	BalanceDays          decimal.Decimal `json:"balanceDays"          bun:"balance_days,type:NUMERIC(8,2),notnull,default:0"`
	AccruedYTDDays       decimal.Decimal `json:"accruedYtdDays"       bun:"accrued_ytd_days,type:NUMERIC(8,2),notnull,default:0"`
	UsedYTDDays          decimal.Decimal `json:"usedYtdDays"          bun:"used_ytd_days,type:NUMERIC(8,2),notnull,default:0"`
	CarriedDays          decimal.Decimal `json:"carriedDays"          bun:"carried_days,type:NUMERIC(8,2),notnull,default:0"`
	EntryCount           int64           `json:"entryCount"           bun:"entry_count,type:BIGINT,notnull"`
	LastAccrualPeriodKey string          `json:"lastAccrualPeriodKey" bun:"last_accrual_period_key,type:VARCHAR(64),nullzero"`
	LastRolloverKey      string          `json:"lastRolloverKey"      bun:"last_rollover_key,type:VARCHAR(64),nullzero"`
	YearStartedAt        *int64          `json:"yearStartedAt"        bun:"year_started_at,type:BIGINT,nullzero"`
	Version              int64           `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64           `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64           `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker *Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (b *WorkerPTOBalance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("wpb_")
		}
		b.CreatedAt = now
		b.UpdatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

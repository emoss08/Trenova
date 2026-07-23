package driverpay

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*EscrowAccount)(nil)
	_ pagination.CursorEntity            = (*EscrowAccount)(nil)
	_ validationframework.TenantedEntity = (*EscrowAccount)(nil)
	_ bun.BeforeAppendModelHook          = (*EscrowTransaction)(nil)
)

type EscrowAccount struct {
	bun.BaseModel             `bun:"table:escrow_accounts,alias:escr" json:"-"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                      pulid.ID            `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID            `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID            `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID                pulid.ID            `json:"workerId"                bun:"worker_id,type:VARCHAR(100),notnull"`
	Status                  EscrowAccountStatus `json:"status"                  bun:"status,type:VARCHAR(50),notnull,default:'Active'"`
	TargetAmountMinor       int64               `json:"targetAmountMinor"       bun:"target_amount_minor,type:BIGINT,notnull,default:0"`
	BalanceMinor            int64               `json:"balanceMinor"            bun:"balance_minor,type:BIGINT,notnull,default:0"`
	AnnualInterestRate      decimal.Decimal     `json:"annualInterestRate"      bun:"annual_interest_rate,type:NUMERIC(7,4),notnull,default:0"`
	LastInterestAccrualDate *int64              `json:"lastInterestAccrualDate" bun:"last_interest_accrual_date,type:BIGINT,nullzero"`
	OpenedDate              int64               `json:"openedDate"              bun:"opened_date,type:BIGINT,notnull"`
	ClosedDate              *int64              `json:"closedDate"              bun:"closed_date,type:BIGINT,nullzero"`
	CurrencyCode            string              `json:"currencyCode"            bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Version                 int64               `json:"version"                 bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt               int64               `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64               `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker       *worker.Worker       `json:"worker,omitempty"       bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Transactions []*EscrowTransaction `json:"transactions,omitempty" bun:"rel:has-many,join:id=escrow_account_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type EscrowTransaction struct {
	bun.BaseModel `bun:"table:escrow_transactions,alias:esctx" json:"-"`

	ID                pulid.ID              `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID              `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID              `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	EscrowAccountID   pulid.ID              `json:"escrowAccountId"   bun:"escrow_account_id,type:VARCHAR(100),notnull"`
	Type              EscrowTransactionType `json:"type"              bun:"type,type:VARCHAR(50),notnull"`
	AmountMinor       int64                 `json:"amountMinor"       bun:"amount_minor,type:BIGINT,notnull"`
	BalanceAfterMinor int64                 `json:"balanceAfterMinor" bun:"balance_after_minor,type:BIGINT,notnull"`
	OccurredDate      int64                 `json:"occurredDate"      bun:"occurred_date,type:BIGINT,notnull"`
	Description       string                `json:"description"       bun:"description,type:VARCHAR(255),nullzero"`
	SettlementID      *pulid.ID             `json:"settlementId"      bun:"settlement_id,type:VARCHAR(100),nullzero"`
	CreatedByID       pulid.ID              `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt         int64                 `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *EscrowAccount) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.OpenedDate, validation.Required.Error("Opened date is required")),
		validation.Field(&e.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if !e.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Escrow account status is invalid")
	}
	if e.TargetAmountMinor < 0 {
		multiErr.Add(
			"targetAmountMinor",
			errortypes.ErrInvalid,
			"Target amount cannot be negative",
		)
	}
	if e.AnnualInterestRate.IsNegative() ||
		e.AnnualInterestRate.GreaterThan(decimal.NewFromInt(100)) {
		multiErr.Add(
			"annualInterestRate",
			errortypes.ErrInvalid,
			"Annual interest rate must be between 0 and 100",
		)
	}
	if e.ClosedDate != nil && *e.ClosedDate < e.OpenedDate {
		multiErr.Add(
			"closedDate",
			errortypes.ErrInvalid,
			"Closed date cannot be before the opened date",
		)
	}
}

func (t *EscrowTransaction) Validate(multiErr *errortypes.MultiError) {
	if !t.Type.IsValid() {
		multiErr.Add("type", errortypes.ErrInvalid, "Escrow transaction type is invalid")
	}
	if t.EscrowAccountID.IsNil() {
		multiErr.Add("escrowAccountId", errortypes.ErrRequired, "Escrow account is required")
	}
	if t.AmountMinor == 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Transaction amount cannot be zero",
		)
	}
	if t.OccurredDate == 0 {
		multiErr.Add("occurredDate", errortypes.ErrRequired, "Occurred date is required")
	}

	switch t.Type {
	case EscrowTransactionTypeContribution, EscrowTransactionTypeInterestAccrual:
		if t.AmountMinor < 0 {
			multiErr.Add(
				"amountMinor",
				errortypes.ErrInvalid,
				"Contributions and interest accruals must be positive",
			)
		}
	case EscrowTransactionTypeApplication, EscrowTransactionTypeRefund:
		if t.AmountMinor > 0 {
			multiErr.Add(
				"amountMinor",
				errortypes.ErrInvalid,
				"Applications and refunds must be negative",
			)
		}
	case EscrowTransactionTypeAdjustment:
	}
}

func (e *EscrowAccount) FundedPercent() decimal.Decimal {
	if e.TargetAmountMinor <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(e.BalanceMinor).
		Div(decimal.NewFromInt(e.TargetAmountMinor)).
		Mul(decimal.NewFromInt(100)).
		Round(2)
}

func (e *EscrowAccount) IsFullyFunded() bool {
	return e.TargetAmountMinor > 0 && e.BalanceMinor >= e.TargetAmountMinor
}

func (e *EscrowAccount) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:       "escr",
		UseSearchVector:  false,
		SearchableFields: []domaintypes.SearchableField{},
	}
}

func (e *EscrowAccount) GetID() pulid.ID { return e.ID }

func (e *EscrowAccount) GetCreatedAt() int64 { return e.CreatedAt }

func (e *EscrowAccount) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *EscrowAccount) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *EscrowAccount) GetTableName() string { return "escrow_accounts" }

func (e *EscrowAccount) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("escr_")
		}
		e.CreatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}
	return nil
}

func (t *EscrowTransaction) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("esctx_")
		}
		t.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

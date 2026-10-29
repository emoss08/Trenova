package payescrow

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ pagination.CursorEntity            = (*EscrowAccount)(nil)
	_ validationframework.TenantedEntity = (*EscrowAccount)(nil)
)

// Owner-operator escrow balances against their target
type EscrowAccount struct {
	keel.Table `keel:"escrow_accounts,report=Payroll"`

	ID                      pulid.ID            `json:"id"                      keel:"id,pk,autoid:escr_"`
	BusinessUnitID          pulid.ID            `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID            `json:"organizationId"          keel:"organization_id,pk,tenant"`
	WorkerID                pulid.ID            `json:"workerId"                keel:"worker_id"`
	Status                  EscrowAccountStatus `json:"status"                  keel:"status"`
	TargetAmountMinor       int64               `json:"targetAmountMinor"       keel:"target_amount_minor,label=Target (Minor)"`
	BalanceMinor            int64               `json:"balanceMinor"            keel:"balance_minor,label=Balance (Minor)"`
	AnnualInterestRate      decimal.Decimal     `json:"annualInterestRate"      keel:"annual_interest_rate,label=Annual Interest Rate (%)"`
	LastInterestAccrualDate *int64              `json:"lastInterestAccrualDate" keel:"last_interest_accrual_date,label=Last Interest Accrual"`
	OpenedDate              int64               `json:"openedDate"              keel:"opened_date,label=Opened"`
	ClosedDate              *int64              `json:"closedDate"              keel:"closed_date,label=Closed"`
	CurrencyCode            string              `json:"currencyCode"            keel:"currency_code,label=Currency"`
	Version                 int64               `json:"version"                 keel:"version,version"`
	CreatedAt               int64               `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64               `json:"updatedAt"               keel:"updated_at,autoupdate"`

	Worker            *worker.Worker       `json:"worker,omitempty"            keel:"belongs:worker_id,label=Driver"`
	Transactions      []*EscrowTransaction `json:"transactions,omitempty"      keel:"many:escrow_account_id,label=Escrow Transactions"`
	LatestTransaction *EscrowTransaction   `json:"latestTransaction,omitempty" keel:"pick:transactions,last=transactions.occurred_date|transactions.created_at"`
}

// Contributions, interest, refunds and deductions against escrow
type EscrowTransaction struct {
	keel.Table `keel:"escrow_transactions,report=Payroll"`

	ID                pulid.ID              `json:"id"                keel:"id,pk,autoid:esctx_"`
	BusinessUnitID    pulid.ID              `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID              `json:"organizationId"    keel:"organization_id,pk,tenant"`
	EscrowAccountID   pulid.ID              `json:"escrowAccountId"   keel:"escrow_account_id"`
	Type              EscrowTransactionType `json:"type"              keel:"type,label=Transaction Type"`
	AmountMinor       int64                 `json:"amountMinor"       keel:"amount_minor,label=Amount (Minor)"`
	BalanceAfterMinor int64                 `json:"balanceAfterMinor" keel:"balance_after_minor,label=Balance After (Minor)"`
	OccurredDate      int64                 `json:"occurredDate"      keel:"occurred_date,label=Occurred"`
	Description       string                `json:"description"       keel:"description,nullzero"`
	SettlementID      *pulid.ID             `json:"settlementId"      keel:"settlement_id"`
	CreatedByID       pulid.ID              `json:"createdById"       keel:"created_by_id,nullzero"`
	CreatedAt         int64                 `json:"createdAt"         keel:"created_at,autocreate"`

	EscrowAccount *EscrowAccount `json:"escrowAccount,omitempty" keel:"belongs:escrow_account_id"`
}

func (e *EscrowAccount) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.OpenedDate, validation.Required.Error("Opened date is required")),
		validation.Field(&e.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

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
		TableAlias:      "escr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "currency_code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (e *EscrowAccount) GetID() pulid.ID { return e.ID }

func (e *EscrowAccount) GetCreatedAt() int64 { return e.CreatedAt }

func (e *EscrowAccount) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *EscrowAccount) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *EscrowAccount) GetTableName() string { return "escrow_accounts" }

func (e *EscrowAccount) BeforeCreate(ctx context.Context) error {
	if e.Status == "" {
		e.Status = EscrowAccountStatusActive
	}
	if e.CurrencyCode == "" {
		e.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (EscrowAccount) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_escrow_accounts_worker_active",
			Message: "Worker already has an active escrow account",
		},
	}
}

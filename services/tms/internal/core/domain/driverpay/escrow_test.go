package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validEscrowAccount() EscrowAccount {
	return EscrowAccount{
		ID:                 pulid.MustNew("escr_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		OrganizationID:     pulid.MustNew("org_"),
		WorkerID:           pulid.MustNew("wrk_"),
		Status:             EscrowAccountStatusActive,
		TargetAmountMinor:  500_000,
		BalanceMinor:       0,
		AnnualInterestRate: decimal.NewFromFloat(2.5),
		OpenedDate:         1_700_000_000,
		CurrencyCode:       "USD",
	}
}

func TestEscrowAccount_Validate(t *testing.T) {
	t.Parallel()

	closedBeforeOpened := int64(1_600_000_000)
	closedAfterOpened := int64(1_800_000_000)

	tests := []struct {
		name    string
		mutate  func(e *EscrowAccount)
		wantErr bool
	}{
		{
			name:    "valid account passes",
			mutate:  func(e *EscrowAccount) {},
			wantErr: false,
		},
		{
			name:    "missing worker fails",
			mutate:  func(e *EscrowAccount) { e.WorkerID = "" },
			wantErr: true,
		},
		{
			name:    "missing opened date fails",
			mutate:  func(e *EscrowAccount) { e.OpenedDate = 0 },
			wantErr: true,
		},
		{
			name:    "missing currency code fails",
			mutate:  func(e *EscrowAccount) { e.CurrencyCode = "" },
			wantErr: true,
		},
		{
			name:    "currency code longer than three characters fails",
			mutate:  func(e *EscrowAccount) { e.CurrencyCode = "USDX" },
			wantErr: true,
		},
		{
			name:    "invalid status fails",
			mutate:  func(e *EscrowAccount) { e.Status = "Frozen" },
			wantErr: true,
		},
		{
			name:    "negative target amount fails",
			mutate:  func(e *EscrowAccount) { e.TargetAmountMinor = -1 },
			wantErr: true,
		},
		{
			name:    "negative interest rate fails",
			mutate:  func(e *EscrowAccount) { e.AnnualInterestRate = decimal.NewFromInt(-1) },
			wantErr: true,
		},
		{
			name:    "interest rate above 100 fails",
			mutate:  func(e *EscrowAccount) { e.AnnualInterestRate = decimal.NewFromFloat(100.01) },
			wantErr: true,
		},
		{
			name:    "closed date before opened date fails",
			mutate:  func(e *EscrowAccount) { e.ClosedDate = &closedBeforeOpened },
			wantErr: true,
		},
		{
			name:    "closed date after opened date passes",
			mutate:  func(e *EscrowAccount) { e.ClosedDate = &closedAfterOpened },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validEscrowAccount()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestEscrowTransaction_Validate(t *testing.T) {
	t.Parallel()

	valid := func() EscrowTransaction {
		return EscrowTransaction{
			ID:              pulid.MustNew("esctx_"),
			BusinessUnitID:  pulid.MustNew("bu_"),
			OrganizationID:  pulid.MustNew("org_"),
			EscrowAccountID: pulid.MustNew("escr_"),
			Type:            EscrowTransactionTypeContribution,
			AmountMinor:     10_000,
			OccurredDate:    1_700_000_000,
		}
	}

	tests := []struct {
		name    string
		mutate  func(tx *EscrowTransaction)
		wantErr bool
	}{
		{
			name:    "valid contribution passes",
			mutate:  func(tx *EscrowTransaction) {},
			wantErr: false,
		},
		{
			name:    "invalid type fails",
			mutate:  func(tx *EscrowTransaction) { tx.Type = "Transfer" },
			wantErr: true,
		},
		{
			name:    "missing escrow account fails",
			mutate:  func(tx *EscrowTransaction) { tx.EscrowAccountID = "" },
			wantErr: true,
		},
		{
			name:    "zero amount fails",
			mutate:  func(tx *EscrowTransaction) { tx.AmountMinor = 0 },
			wantErr: true,
		},
		{
			name:    "missing occurred date fails",
			mutate:  func(tx *EscrowTransaction) { tx.OccurredDate = 0 },
			wantErr: true,
		},
		{
			name:    "negative contribution fails",
			mutate:  func(tx *EscrowTransaction) { tx.AmountMinor = -1 },
			wantErr: true,
		},
		{
			name: "negative interest accrual fails",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeInterestAccrual
				tx.AmountMinor = -1
			},
			wantErr: true,
		},
		{
			name: "positive application fails",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeApplication
				tx.AmountMinor = 500
			},
			wantErr: true,
		},
		{
			name: "negative application passes",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeApplication
				tx.AmountMinor = -500
			},
			wantErr: false,
		},
		{
			name: "positive refund fails",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeRefund
				tx.AmountMinor = 500
			},
			wantErr: true,
		},
		{
			name: "negative refund passes",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeRefund
				tx.AmountMinor = -500
			},
			wantErr: false,
		},
		{
			name: "positive adjustment passes",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeAdjustment
				tx.AmountMinor = 500
			},
			wantErr: false,
		},
		{
			name: "negative adjustment passes",
			mutate: func(tx *EscrowTransaction) {
				tx.Type = EscrowTransactionTypeAdjustment
				tx.AmountMinor = -500
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := valid()
			tt.mutate(&tx)
			multiErr := errortypes.NewMultiError()
			tx.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestEscrowAccount_FundedPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  int64
		balance int64
		want    string
	}{
		{name: "zero target returns zero", target: 0, balance: 100, want: "0"},
		{name: "negative target returns zero", target: -100, balance: 100, want: "0"},
		{name: "half funded", target: 500_000, balance: 250_000, want: "50"},
		{name: "fully funded", target: 500_000, balance: 500_000, want: "100"},
		{name: "over funded exceeds 100", target: 500_000, balance: 750_000, want: "150"},
		{name: "rounds to two decimals", target: 300, balance: 100, want: "33.33"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			account := EscrowAccount{TargetAmountMinor: tt.target, BalanceMinor: tt.balance}
			assert.Equal(t, tt.want, account.FundedPercent().String())
		})
	}
}

func TestEscrowAccount_IsFullyFunded(t *testing.T) {
	t.Parallel()

	assert.False(t, (&EscrowAccount{TargetAmountMinor: 0, BalanceMinor: 100}).IsFullyFunded())
	assert.False(t, (&EscrowAccount{TargetAmountMinor: 500, BalanceMinor: 499}).IsFullyFunded())
	assert.True(t, (&EscrowAccount{TargetAmountMinor: 500, BalanceMinor: 500}).IsFullyFunded())
	assert.True(t, (&EscrowAccount{TargetAmountMinor: 500, BalanceMinor: 600}).IsFullyFunded())
}

package driverpayservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEscrowEntity() *driverpay.EscrowAccount {
	return &driverpay.EscrowAccount{
		BusinessUnitID:    pulid.MustNew("bu_"),
		OrganizationID:    pulid.MustNew("org_"),
		WorkerID:          pulid.MustNew("wrk_"),
		TargetAmountMinor: 500_000,
		OpenedDate:        timeutils.NowUnix(),
		CurrencyCode:      "USD",
	}
}

func activeEscrowAccount(balance int64) *driverpay.EscrowAccount {
	account := newEscrowEntity()
	account.ID = pulid.MustNew("escr_")
	account.Status = driverpay.EscrowAccountStatusActive
	account.BalanceMinor = balance
	return account
}

func TestOpenEscrowAccount_RequiresActor(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	_, err := svc.OpenEscrowAccount(t.Context(), newEscrowEntity(), nil)
	require.Error(t, err)

	_, err = svc.OpenEscrowAccount(t.Context(), newEscrowEntity(), &serviceports.RequestActor{})
	require.Error(t, err)
}

func TestOpenEscrowAccount_DefaultsInterestRateFromSettlementControl(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	var created *driverpay.EscrowAccount
	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, repositories.GetActiveEscrowAccountForWorkerRequest) (*driverpay.EscrowAccount, error) {
			return nil, errors.New("not found")
		},
		create: func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
			entity.ID = pulid.MustNew("escr_")
			created = entity
			return entity, nil
		},
	}
	svc.settlementControl = &fakeSettlementControlRepo{
		getOrCreate: func(context.Context, pagination.TenantInfo) (*tenant.SettlementControl, error) {
			return &tenant.SettlementControl{
				DefaultEscrowInterestRate: decimal.NewFromFloat(3.25),
			}, nil
		},
	}

	entity := newEscrowEntity()
	entity.Status = driverpay.EscrowAccountStatusClosed
	entity.BalanceMinor = 999

	result, err := svc.OpenEscrowAccount(t.Context(), entity, testActor())
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, driverpay.EscrowAccountStatusActive, result.Status)
	assert.EqualValues(t, 0, result.BalanceMinor)
	assert.Equal(t, "3.25", result.AnnualInterestRate.String())
	assert.Len(t, audit.logged, 1)
}

func TestOpenEscrowAccount_RejectsDuplicateActiveAccount(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, repositories.GetActiveEscrowAccountForWorkerRequest) (*driverpay.EscrowAccount, error) {
			return activeEscrowAccount(0), nil
		},
	}

	entity := newEscrowEntity()
	entity.AnnualInterestRate = decimal.NewFromInt(2)

	_, err := svc.OpenEscrowAccount(t.Context(), entity, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active escrow account")
	assert.Empty(t, audit.logged)
}

func TestAdjustEscrowAccount_ValidatesRequest(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	_, err := svc.AdjustEscrowAccount(t.Context(), &EscrowAdjustmentRequest{
		TenantInfo:  tenantInfo,
		AccountID:   pulid.MustNew("escr_"),
		AmountMinor: 0,
		Description: "zero",
	}, testActor())
	require.Error(t, err)

	_, err = svc.AdjustEscrowAccount(t.Context(), &EscrowAdjustmentRequest{
		TenantInfo:  tenantInfo,
		AccountID:   pulid.MustNew("escr_"),
		AmountMinor: 1_000,
	}, testActor())
	require.Error(t, err)
}

func TestAdjustEscrowAccount_RejectsInactiveAccount(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	account := activeEscrowAccount(10_000)
	account.Status = driverpay.EscrowAccountStatusClosed
	svc.escrowRepo = &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}

	_, err := svc.AdjustEscrowAccount(t.Context(), &EscrowAdjustmentRequest{
		AccountID:   account.ID,
		AmountMinor: 1_000,
		Description: "adjustment",
	}, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestAdjustEscrowAccount_RejectsNegativeResultingBalance(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	account := activeEscrowAccount(5_000)
	svc.escrowRepo = &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}

	_, err := svc.AdjustEscrowAccount(t.Context(), &EscrowAdjustmentRequest{
		AccountID:   account.ID,
		AmountMinor: -5_001,
		Description: "drain",
	}, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestAdjustEscrowAccount_AppliesTransactionAndUpdatesBalance(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	account := activeEscrowAccount(5_000)
	repo := &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	var appended *driverpay.EscrowTransaction
	repo.update = func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
		return entity, nil
	}
	repo.appendTransaction = func(_ context.Context, entity *driverpay.EscrowTransaction) (*driverpay.EscrowTransaction, error) {
		appended = entity
		return entity, nil
	}
	svc.escrowRepo = repo

	updated, err := svc.AdjustEscrowAccount(t.Context(), &EscrowAdjustmentRequest{
		AccountID:   account.ID,
		AmountMinor: -2_000,
		Description: "damage claim applied",
	}, testActor())
	require.NoError(t, err)
	require.NotNil(t, appended)

	assert.Equal(t, driverpay.EscrowTransactionTypeAdjustment, appended.Type)
	assert.EqualValues(t, -2_000, appended.AmountMinor)
	assert.EqualValues(t, 3_000, appended.BalanceAfterMinor)
	assert.Equal(t, account.ID, appended.EscrowAccountID)
	assert.Equal(t, account.OrganizationID, appended.OrganizationID)
	assert.EqualValues(t, 3_000, updated.BalanceMinor)
	assert.Len(t, audit.logged, 1)
}

func TestCloseEscrowAccount_RejectsAlreadyClosedAccount(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	account := activeEscrowAccount(0)
	account.Status = driverpay.EscrowAccountStatusClosed
	svc.escrowRepo = &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}

	_, err := svc.CloseEscrowAccount(
		t.Context(),
		pagination.TenantInfo{},
		account.ID,
		testActor(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already closed")
}

func TestCloseEscrowAccount_RefundsRemainingBalance(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	account := activeEscrowAccount(75_000)
	repo := &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	var appended *driverpay.EscrowTransaction
	repo.update = func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
		return entity, nil
	}
	repo.appendTransaction = func(_ context.Context, entity *driverpay.EscrowTransaction) (*driverpay.EscrowTransaction, error) {
		appended = entity
		return entity, nil
	}
	svc.escrowRepo = repo

	updated, err := svc.CloseEscrowAccount(
		t.Context(),
		pagination.TenantInfo{},
		account.ID,
		testActor(),
	)
	require.NoError(t, err)
	require.NotNil(t, appended)

	assert.Equal(t, driverpay.EscrowTransactionTypeRefund, appended.Type)
	assert.EqualValues(t, -75_000, appended.AmountMinor)
	assert.EqualValues(t, 0, appended.BalanceAfterMinor)
	assert.Equal(t, driverpay.EscrowAccountStatusClosed, updated.Status)
	require.NotNil(t, updated.ClosedDate)
	assert.EqualValues(t, 0, updated.BalanceMinor)
	assert.Len(t, audit.logged, 1)
}

func TestCloseEscrowAccount_ZeroBalanceSkipsRefund(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	account := activeEscrowAccount(0)
	appendCalls := 0
	repo := &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	repo.update = func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
		return entity, nil
	}
	repo.appendTransaction = func(_ context.Context, entity *driverpay.EscrowTransaction) (*driverpay.EscrowTransaction, error) {
		appendCalls++
		return entity, nil
	}
	svc.escrowRepo = repo

	updated, err := svc.CloseEscrowAccount(
		t.Context(),
		pagination.TenantInfo{},
		account.ID,
		testActor(),
	)
	require.NoError(t, err)
	assert.Equal(t, 0, appendCalls)
	assert.Equal(t, driverpay.EscrowAccountStatusClosed, updated.Status)
}

func TestAccrueEscrowInterest_SkipsIneligibleAccounts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(a *driverpay.EscrowAccount)
	}{
		{
			name:   "closed account",
			mutate: func(a *driverpay.EscrowAccount) { a.Status = driverpay.EscrowAccountStatusClosed },
		},
		{
			name:   "zero balance",
			mutate: func(a *driverpay.EscrowAccount) { a.BalanceMinor = 0 },
		},
		{
			name:   "zero rate",
			mutate: func(a *driverpay.EscrowAccount) { a.AnnualInterestRate = decimal.Zero },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc, _ := newTestService()

			account := activeEscrowAccount(100_000)
			account.AnnualInterestRate = decimal.NewFromInt(5)
			tc.mutate(account)

			svc.escrowRepo = &fakeEscrowRepo{
				getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
					return account, nil
				},
			}

			result, err := svc.AccrueEscrowInterest(
				t.Context(),
				pagination.TenantInfo{},
				account.ID,
			)
			require.NoError(t, err)
			assert.Same(t, account, result)
		})
	}
}

func TestAccrueEscrowInterest_AccruesOneYearOfInterest(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour).Unix()
	account := activeEscrowAccount(1_000_000)
	account.AnnualInterestRate = decimal.NewFromInt(5)
	account.LastInterestAccrualDate = &oneYearAgo

	repo := &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	var appended *driverpay.EscrowTransaction
	repo.update = func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
		return entity, nil
	}
	repo.appendTransaction = func(_ context.Context, entity *driverpay.EscrowTransaction) (*driverpay.EscrowTransaction, error) {
		appended = entity
		return entity, nil
	}
	svc.escrowRepo = repo

	updated, err := svc.AccrueEscrowInterest(t.Context(), pagination.TenantInfo{}, account.ID)
	require.NoError(t, err)
	require.NotNil(t, appended)

	assert.Equal(t, driverpay.EscrowTransactionTypeInterestAccrual, appended.Type)
	assert.EqualValues(t, 50_000, appended.AmountMinor)
	assert.EqualValues(t, 1_050_000, updated.BalanceMinor)
	require.NotNil(t, updated.LastInterestAccrualDate)
	assert.Greater(t, *updated.LastInterestAccrualDate, oneYearAgo)
}

func TestAccrueEscrowInterest_TinyElapsedOnlyAdvancesAccrualDate(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	justNow := timeutils.NowUnix() - 1
	account := activeEscrowAccount(1_000)
	account.AnnualInterestRate = decimal.NewFromInt(5)
	account.LastInterestAccrualDate = &justNow

	appendCalls := 0
	repo := &fakeEscrowRepo{
		getByID: func(context.Context, repositories.GetEscrowAccountByIDRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	repo.update = func(_ context.Context, entity *driverpay.EscrowAccount) (*driverpay.EscrowAccount, error) {
		return entity, nil
	}
	repo.appendTransaction = func(_ context.Context, entity *driverpay.EscrowTransaction) (*driverpay.EscrowTransaction, error) {
		appendCalls++
		return entity, nil
	}
	svc.escrowRepo = repo

	updated, err := svc.AccrueEscrowInterest(t.Context(), pagination.TenantInfo{}, account.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, appendCalls)
	assert.EqualValues(t, 1_000, updated.BalanceMinor)
	require.NotNil(t, updated.LastInterestAccrualDate)
	assert.GreaterOrEqual(t, *updated.LastInterestAccrualDate, justNow)
}

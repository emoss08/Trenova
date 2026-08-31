package driverpayservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePayCodeRepo struct {
	repositories.PayCodeRepository
	getByID func(
		ctx context.Context,
		req repositories.GetPayCodeByIDRequest,
	) (*driverpay.PayCode, error)
}

func (f *fakePayCodeRepo) GetByID(
	ctx context.Context,
	req repositories.GetPayCodeByIDRequest,
) (*driverpay.PayCode, error) {
	return f.getByID(ctx, req)
}

type fakeDeductionRepo struct {
	repositories.RecurringDeductionRepository
	create func(
		ctx context.Context,
		entity *driverpay.RecurringDeduction,
	) (*driverpay.RecurringDeduction, error)
}

func (f *fakeDeductionRepo) Create(
	ctx context.Context,
	entity *driverpay.RecurringDeduction,
) (*driverpay.RecurringDeduction, error) {
	return f.create(ctx, entity)
}

func newDeductionEntity() *driverpay.RecurringDeduction {
	return &driverpay.RecurringDeduction{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayCodeID:      pulid.MustNew("payc_"),
		Status:         driverpay.DeductionStatusActive,
		Frequency:      driverpay.DeductionFrequencyEverySettlement,
		Description:    "Escrow contribution",
		AmountMinor:    25_000,
		StartDate:      timeutils.NowUnix(),
		CurrencyCode:   "USD",
	}
}

func deductionPayCode(direction driverpay.PayCodeDirection) *driverpay.PayCode {
	return &driverpay.PayCode{
		ID:        pulid.MustNew("payc_"),
		Direction: direction,
		Code:      "ESCROW",
		Name:      "Escrow Contribution",
	}
}

func TestCreateDeduction_RejectsEarningPayCode(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	svc.payCodeRepo = &fakePayCodeRepo{
		getByID: func(context.Context, repositories.GetPayCodeByIDRequest) (*driverpay.PayCode, error) {
			return deductionPayCode(driverpay.PayCodeDirectionEarning), nil
		},
	}

	_, err := svc.CreateDeduction(t.Context(), newDeductionEntity(), false, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used here")
}

func TestCreateDeduction_AutoLinksActiveEscrowAccount(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	account := activeEscrowAccount(0)
	svc.payCodeRepo = &fakePayCodeRepo{
		getByID: func(context.Context, repositories.GetPayCodeByIDRequest) (*driverpay.PayCode, error) {
			return deductionPayCode(driverpay.PayCodeDirectionDeduction), nil
		},
	}
	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, repositories.GetActiveEscrowAccountForWorkerRequest) (*driverpay.EscrowAccount, error) {
			return account, nil
		},
	}
	svc.deductionRepo = &fakeDeductionRepo{
		create: func(_ context.Context, entity *driverpay.RecurringDeduction) (*driverpay.RecurringDeduction, error) {
			entity.ID = pulid.MustNew("rded_")
			return entity, nil
		},
	}

	actor := testActor()
	created, err := svc.CreateDeduction(t.Context(), newDeductionEntity(), true, actor)
	require.NoError(t, err)

	require.NotNil(t, created.EscrowAccountID)
	assert.Equal(t, account.ID, *created.EscrowAccountID)
	assert.True(t, created.IsEscrowContribution())
	assert.Equal(t, actor.UserID, created.CreatedByID)
	assert.Len(t, audit.logged, 1)
}

func TestCreateDeduction_AutoLinkFailsWithoutActiveEscrowAccount(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	svc.payCodeRepo = &fakePayCodeRepo{
		getByID: func(context.Context, repositories.GetPayCodeByIDRequest) (*driverpay.PayCode, error) {
			return deductionPayCode(driverpay.PayCodeDirectionDeduction), nil
		},
	}
	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, repositories.GetActiveEscrowAccountForWorkerRequest) (*driverpay.EscrowAccount, error) {
			return nil, errors.New("not found")
		},
	}

	_, err := svc.CreateDeduction(t.Context(), newDeductionEntity(), true, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active escrow account")
}

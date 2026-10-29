package driverpayservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePayCodeRepo struct {
	driverpayports.PayCodeRepository
	getByID func(
		ctx context.Context,
		req driverpayports.GetPayCodeByIDRequest,
	) (*paycode.PayCode, error)
}

func (f *fakePayCodeRepo) GetByID(
	ctx context.Context,
	req driverpayports.GetPayCodeByIDRequest,
) (*paycode.PayCode, error) {
	return f.getByID(ctx, req)
}

type fakeDeductionRepo struct {
	driverpayports.RecurringDeductionRepository
	create func(
		ctx context.Context,
		entity *recurringpay.RecurringDeduction,
	) (*recurringpay.RecurringDeduction, error)
}

func (f *fakeDeductionRepo) Create(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
) (*recurringpay.RecurringDeduction, error) {
	return f.create(ctx, entity)
}

func newDeductionEntity() *recurringpay.RecurringDeduction {
	return &recurringpay.RecurringDeduction{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayCodeID:      pulid.MustNew("payc_"),
		Status:         recurringpay.DeductionStatusActive,
		Frequency:      recurringpay.DeductionFrequencyEverySettlement,
		Description:    "Escrow contribution",
		AmountMinor:    25_000,
		StartDate:      timeutils.NowUnix(),
		CurrencyCode:   "USD",
	}
}

func deductionPayCode(direction paycode.PayCodeDirection) *paycode.PayCode {
	return &paycode.PayCode{
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
		getByID: func(context.Context, driverpayports.GetPayCodeByIDRequest) (*paycode.PayCode, error) {
			return deductionPayCode(paycode.PayCodeDirectionEarning), nil
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
		getByID: func(context.Context, driverpayports.GetPayCodeByIDRequest) (*paycode.PayCode, error) {
			return deductionPayCode(paycode.PayCodeDirectionDeduction), nil
		},
	}
	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, driverpayports.GetActiveEscrowAccountForWorkerRequest) (*payescrow.EscrowAccount, error) {
			return account, nil
		},
	}
	svc.deductionRepo = &fakeDeductionRepo{
		create: func(_ context.Context, entity *recurringpay.RecurringDeduction) (*recurringpay.RecurringDeduction, error) {
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
		getByID: func(context.Context, driverpayports.GetPayCodeByIDRequest) (*paycode.PayCode, error) {
			return deductionPayCode(paycode.PayCodeDirectionDeduction), nil
		},
	}
	svc.escrowRepo = &fakeEscrowRepo{
		getActiveForWorker: func(context.Context, driverpayports.GetActiveEscrowAccountForWorkerRequest) (*payescrow.EscrowAccount, error) {
			return nil, errors.New("not found")
		},
	}

	_, err := svc.CreateDeduction(t.Context(), newDeductionEntity(), true, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active escrow account")
}

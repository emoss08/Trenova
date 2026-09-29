package ptoledgerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type balanceReader struct {
	repositories.PTOLedgerRepository
	balances []*worker.WorkerPTOBalance
}

func (r *balanceReader) ListBalances(
	context.Context,
	*repositories.ListPTOBalancesRequest,
) ([]*worker.WorkerPTOBalance, error) {
	return r.balances, nil
}

func TestPlanAdjust_ProjectsTheBalance(t *testing.T) {
	t.Parallel()

	svc := &Service{ledgerRepo: &balanceReader{balances: []*worker.WorkerPTOBalance{
		{PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(4)},
		{PTOType: worker.PTOTypeSick, BalanceDays: decimal.NewFromInt(2)},
	}}}
	req := &AdjustRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		WorkerID:   pulid.MustNew("wrk_"),
		PTOType:    worker.PTOTypeVacation,
		AmountDays: decimal.NewFromFloat(1.5),
		Note:       "Carried over from the old system",
		UserID:     pulid.MustNew("usr_"),
	}

	plan, err := svc.PlanAdjust(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.BalanceBefore.Equal(decimal.NewFromInt(4)))
	assert.True(t, plan.BalanceAfter.Equal(decimal.NewFromFloat(5.5)))
	assert.Equal(t, worker.PTOLedgerEntryAdjustment, plan.Entry.EntryType)

	req.Note = ""
	_, err = svc.PlanAdjust(t.Context(), req)
	require.Error(t, err, "an adjustment needs a note")

	req.Note = "x"
	req.AmountDays = decimal.Zero
	_, err = svc.PlanAdjust(t.Context(), req)
	require.Error(t, err)
}

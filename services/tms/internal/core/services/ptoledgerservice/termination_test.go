package ptoledgerservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func settlementPolicy() *worker.PTOPolicy {
	return &worker.PTOPolicy{
		Rules: []*worker.PTOPolicyRule{
			{PTOType: worker.PTOTypeVacation, OnTermination: worker.PTOTerminationPayOut},
			{PTOType: worker.PTOTypeSick},
		},
	}
}

func TestPlanTerminationSettlement(t *testing.T) {
	workerID := pulid.MustNew("wrk_")
	balances := []*worker.WorkerPTOBalance{
		{WorkerID: workerID, PTOType: worker.PTOTypeSick, BalanceDays: decimal.NewFromFloat(3.5)},
		{WorkerID: workerID, PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(12)},
		{WorkerID: workerID, PTOType: worker.PTOTypePersonal, BalanceDays: decimal.NewFromInt(2)},
		{WorkerID: workerID, PTOType: worker.PTOTypeBereavement, BalanceDays: decimal.NewFromInt(-1)},
	}

	plan := ptoledgerservice.PlanTerminationSettlement(balances, settlementPolicy())

	require.Len(t, plan, 2, "untracked types and negative balances are left alone")
	assert.Equal(t, worker.PTOTypeSick, plan[0].PTOType)
	assert.Equal(t, worker.PTOLedgerEntryForfeiture, plan[0].EntryType)
	assert.True(t, plan[0].Days.Equal(decimal.NewFromFloat(3.5)))
	assert.Equal(t, worker.PTOTypeVacation, plan[1].PTOType)
	assert.Equal(t, worker.PTOLedgerEntryPayout, plan[1].EntryType)
	assert.True(t, plan[1].Days.Equal(decimal.NewFromInt(12)))

	assert.Empty(t, ptoledgerservice.PlanTerminationSettlement(balances, nil), "no policy, nothing to settle")
}

func TestAggregateLiability(t *testing.T) {
	alice := pulid.MustNew("wrk_")
	bob := pulid.MustNew("wrk_")
	carol := pulid.MustNew("wrk_")
	balances := []*worker.WorkerPTOBalance{
		{WorkerID: alice, PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(5)},
		{WorkerID: alice, PTOType: worker.PTOTypeSick, BalanceDays: decimal.NewFromInt(4)},
		{WorkerID: bob, PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(9)},
		{WorkerID: bob, PTOType: worker.PTOTypeSick, BalanceDays: decimal.NewFromInt(-2)},
		{WorkerID: carol, PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(7)},
	}
	policies := map[pulid.ID]*worker.PTOPolicy{alice: settlementPolicy(), bob: settlementPolicy()}

	report := ptoledgerservice.AggregateLiability(1_800_000_000, balances, policies)

	assert.Equal(t, int64(1_800_000_000), report.AsOf)
	assert.Equal(t, 3, report.WorkersTracked)
	assert.True(t, report.TotalBalanceDays.Equal(decimal.NewFromInt(23)), report.TotalBalanceDays.String())
	assert.True(t, report.LiabilityDays.Equal(decimal.NewFromInt(14)), "only pay-out rules owe money")
	assert.True(t, report.ForfeitableDays.Equal(decimal.NewFromInt(11)), "sick days and unpoliced workers forfeit")

	require.Len(t, report.Rows, 5)
	assert.Equal(t, bob, report.Rows[0].WorkerID, "largest liability first")
	assert.True(t, report.Rows[0].LiabilityDays.Equal(decimal.NewFromInt(9)))
	assert.Equal(t, alice, report.Rows[1].WorkerID)
	assert.Equal(t, worker.PTOTypeVacation, report.Rows[1].PTOType)
	assert.Equal(t, carol, report.Rows[2].WorkerID, "then by balance")
	assert.Equal(t, worker.PTOTerminationForfeit, report.Rows[2].OnTermination)
	assert.True(t, report.Rows[2].LiabilityDays.IsZero())
	assert.Equal(t, worker.PTOTypeSick, report.Rows[4].PTOType)
	assert.Equal(t, bob, report.Rows[4].WorkerID, "negative balance sorts last")
}

package driversettlement

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncTotalsBalancedSettlement(t *testing.T) {
	s := &Settlement{
		Lines: []*SettlementLine{
			{Category: LineCategoryEarning, AmountMinor: 100000},
			{Category: LineCategoryReimbursement, AmountMinor: 5000},
			{Category: LineCategoryDeduction, AmountMinor: -20000},
			{Category: LineCategoryAdvanceRecovery, AmountMinor: -10000},
			{Category: LineCategoryEscrowContribution, AmountMinor: -5000},
		},
	}
	s.SyncTotals()

	assert.Equal(t, int64(100000), s.GrossEarningsMinor)
	assert.Equal(t, int64(5000), s.ReimbursementsMinor)
	assert.Equal(t, int64(35000), s.DeductionsMinor)
	assert.Equal(t, int64(70000), s.NetPayMinor)
	assert.Zero(t, s.CarryForwardOutMinor)
}

func TestSyncTotalsNegativeNetCarriesForward(t *testing.T) {
	s := &Settlement{
		Lines: []*SettlementLine{
			{Category: LineCategoryEarning, AmountMinor: 20000},
			{Category: LineCategoryDeduction, AmountMinor: -50000},
		},
	}
	s.SyncTotals()

	assert.Zero(t, s.NetPayMinor)
	assert.Equal(t, int64(-30000), s.CarryForwardOutMinor)
}

func TestSyncTotalsAppliesCarryForwardIn(t *testing.T) {
	s := &Settlement{
		Lines: []*SettlementLine{
			{Category: LineCategoryEarning, AmountMinor: 100000},
			{Category: LineCategoryCarryForward, AmountMinor: -30000},
		},
	}
	s.SyncTotals()

	assert.Equal(t, int64(100000), s.GrossEarningsMinor)
	assert.Equal(t, int64(-30000), s.CarryForwardInMinor)
	assert.Equal(t, int64(70000), s.NetPayMinor)
}

func TestSyncTotalsGuaranteeTopUpCountsAsGross(t *testing.T) {
	s := &Settlement{
		Lines: []*SettlementLine{
			{Category: LineCategoryEarning, AmountMinor: 80000},
			{Category: LineCategoryGuaranteeTopUp, AmountMinor: 20000},
		},
	}
	s.SyncTotals()

	assert.Equal(t, int64(100000), s.GrossEarningsMinor)
	assert.Equal(t, int64(100000), s.NetPayMinor)
}

func TestSyncTotalsAssignsLineNumbers(t *testing.T) {
	s := &Settlement{
		Lines: []*SettlementLine{
			{Category: LineCategoryEarning, AmountMinor: 1000},
			{Category: LineCategoryDeduction, AmountMinor: -100},
		},
	}
	s.SyncTotals()

	assert.Equal(t, 1, s.Lines[0].LineNumber)
	assert.Equal(t, 2, s.Lines[1].LineNumber)
}

func TestStatusTransitions(t *testing.T) {
	assert.True(t, IsAllowedTransition(StatusDraft, StatusPendingApproval))
	assert.True(t, IsAllowedTransition(StatusPendingApproval, StatusApproved))
	assert.True(t, IsAllowedTransition(StatusPendingApproval, StatusDraft))
	assert.True(t, IsAllowedTransition(StatusApproved, StatusPosted))
	assert.True(t, IsAllowedTransition(StatusPosted, StatusPaid))
	assert.True(t, IsAllowedTransition(StatusPosted, StatusVoided))

	assert.False(t, IsAllowedTransition(StatusDraft, StatusPosted))
	assert.False(t, IsAllowedTransition(StatusPaid, StatusDraft))
	assert.False(t, IsAllowedTransition(StatusVoided, StatusDraft))

	assert.True(t, IsTerminalStatus(StatusPaid))
	assert.True(t, IsTerminalStatus(StatusVoided))
	assert.False(t, IsTerminalStatus(StatusApproved))
}

func TestAddExceptionDeduplicates(t *testing.T) {
	s := &Settlement{}
	s.AddException(ExceptionCodeNegativeNet, ExceptionSeverityCritical, "first")
	s.AddException(ExceptionCodeNegativeNet, ExceptionSeverityCritical, "second")

	assert.Len(t, s.Exceptions, 1)
	assert.True(t, s.HasExceptions)
}

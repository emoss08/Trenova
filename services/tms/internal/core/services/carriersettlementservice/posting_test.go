package carriersettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	expenseAccountID  = pulid.ID("gla_expense")
	overrideAccountID = pulid.ID("gla_override")
	payableAccountID  = pulid.ID("gla_payable")
	cashAccountID     = pulid.ID("gla_cash")
)

func newPostingSettlement() *carriersettlement.CarrierSettlement {
	entity := &carriersettlement.CarrierSettlement{
		ID:             pulid.MustNew("carstl_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CarrierID:      pulid.MustNew("car_"),
		Status:         carriersettlement.StatusApproved,
		Lines: []*carriersettlement.CarrierSettlementLine{
			{EventType: carriersettlement.CostEventTypeLinehaulCost, AmountMinor: 150_000},
			{EventType: carriersettlement.CostEventTypeFuelSurcharge, AmountMinor: 20_000},
			{EventType: carriersettlement.CostEventTypeAccessorial, AmountMinor: 5_000},
		},
	}
	entity.SyncTotals()
	return entity
}

func assertBalanced(t *testing.T, legs []PostingLeg) {
	t.Helper()
	var debits, credits int64
	for _, leg := range legs {
		assert.False(t, leg.Debit != 0 && leg.Credit != 0,
			"a leg must not carry both a debit and a credit")
		debits += leg.Debit
		credits += leg.Credit
	}
	assert.Equal(t, debits, credits, "total debits must equal total credits")
}

func TestBuildCarrierSettlementPostingLegsGolden(t *testing.T) {
	entity := newPostingSettlement()
	legs := BuildCarrierSettlementPostingLegs(entity, &PostingAccounts{
		Expense: expenseAccountID,
		Payable: payableAccountID,
	})

	require.Len(t, legs, 2)
	assert.Equal(t, PostingLeg{AccountID: expenseAccountID, Debit: 175_000}, legs[0])
	assert.Equal(t, PostingLeg{AccountID: payableAccountID, Credit: 175_000}, legs[1])
	assertBalanced(t, legs)
}

func TestBuildCarrierSettlementPostingLegsWithOverridesAndAdjustments(t *testing.T) {
	entity := newPostingSettlement()
	override := overrideAccountID
	entity.Lines = append(entity.Lines,
		&carriersettlement.CarrierSettlementLine{
			EventType:   carriersettlement.CostEventTypeAdjustment,
			AmountMinor: -25_000,
			GLAccountID: &override,
		},
		&carriersettlement.CarrierSettlementLine{
			EventType:   carriersettlement.CostEventTypeAdjustment,
			AmountMinor: 10_000,
		},
	)
	entity.SyncTotals()
	require.Equal(t, int64(160_000), entity.NetPayableMinor)

	legs := BuildCarrierSettlementPostingLegs(entity, &PostingAccounts{
		Expense: expenseAccountID,
		Payable: payableAccountID,
	})

	require.Len(t, legs, 3)
	assert.Equal(t, PostingLeg{AccountID: expenseAccountID, Debit: 185_000}, legs[0])
	assert.Equal(t, PostingLeg{AccountID: overrideAccountID, Credit: 25_000}, legs[1])
	assert.Equal(t, PostingLeg{AccountID: payableAccountID, Credit: 160_000}, legs[2])
	assertBalanced(t, legs)
}

func TestBuildCarrierSettlementPostingLegsNegativeNetDebitsAP(t *testing.T) {
	entity := &carriersettlement.CarrierSettlement{
		Lines: []*carriersettlement.CarrierSettlementLine{
			{EventType: carriersettlement.CostEventTypeAdjustment, AmountMinor: -40_000},
		},
	}
	entity.SyncTotals()

	legs := BuildCarrierSettlementPostingLegs(entity, &PostingAccounts{
		Expense: expenseAccountID,
		Payable: payableAccountID,
	})

	require.Len(t, legs, 2)
	assert.Equal(t, PostingLeg{AccountID: expenseAccountID, Credit: 40_000}, legs[0])
	assert.Equal(t, PostingLeg{AccountID: payableAccountID, Debit: 40_000}, legs[1])
	assertBalanced(t, legs)
}

func TestBuildCarrierSettlementPostingLegsZeroAmount(t *testing.T) {
	entity := &carriersettlement.CarrierSettlement{}
	entity.SyncTotals()
	legs := BuildCarrierSettlementPostingLegs(entity, &PostingAccounts{
		Expense: expenseAccountID,
		Payable: payableAccountID,
	})
	assert.Empty(t, legs)
}

func TestReverseLegsSwapsDebitsAndCredits(t *testing.T) {
	entity := newPostingSettlement()
	legs := BuildCarrierSettlementPostingLegs(entity, &PostingAccounts{
		Expense: expenseAccountID,
		Payable: payableAccountID,
	})
	reversed := ReverseLegs(legs)

	require.Len(t, reversed, len(legs))
	for idx := range legs {
		assert.Equal(t, legs[idx].AccountID, reversed[idx].AccountID)
		assert.Equal(t, legs[idx].Debit, reversed[idx].Credit)
		assert.Equal(t, legs[idx].Credit, reversed[idx].Debit)
	}
	assertBalanced(t, reversed)
}

func TestBuildCarrierSettlementPaymentLegs(t *testing.T) {
	entity := newPostingSettlement()
	legs := BuildCarrierSettlementPaymentLegs(entity, payableAccountID, cashAccountID)

	require.Len(t, legs, 2)
	assert.Equal(t, PostingLeg{AccountID: payableAccountID, Debit: 175_000}, legs[0])
	assert.Equal(t, PostingLeg{AccountID: cashAccountID, Credit: 175_000}, legs[1])
	assertBalanced(t, legs)

	entity.NetPayableMinor = 0
	assert.Empty(t, BuildCarrierSettlementPaymentLegs(entity, payableAccountID, cashAccountID))

	entity.NetPayableMinor = -30_000
	legs = BuildCarrierSettlementPaymentLegs(entity, payableAccountID, cashAccountID)
	require.Len(t, legs, 2)
	assert.Equal(t, PostingLeg{AccountID: cashAccountID, Debit: 30_000}, legs[0])
	assert.Equal(t, PostingLeg{AccountID: payableAccountID, Credit: 30_000}, legs[1])
	assertBalanced(t, legs)
}

func TestResolvePostingAccounts(t *testing.T) {
	entity := newPostingSettlement()

	t.Run("carrier control defaults win over accounting control", func(t *testing.T) {
		carrierExpense := overrideAccountID
		carrierPayable := pulid.ID("gla_carrier_ap")
		resolved, err := ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{
			DefaultPurchasedTransportationAccountID: &carrierExpense,
			DefaultAPAccountID:                      &carrierPayable,
		}, &tenant.AccountingControl{
			DefaultPurchasedTransportationAccountID: expenseAccountID,
			DefaultAPAccountID:                      payableAccountID,
			DefaultCashAccountID:                    cashAccountID,
		}, false)
		require.NoError(t, err)
		assert.Equal(t, carrierExpense, resolved.Expense)
		assert.Equal(t, carrierPayable, resolved.Payable)
	})

	t.Run("falls back to accounting control", func(t *testing.T) {
		resolved, err := ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{},
			&tenant.AccountingControl{
				DefaultPurchasedTransportationAccountID: expenseAccountID,
				DefaultAPAccountID:                      payableAccountID,
				DefaultCashAccountID:                    cashAccountID,
			}, true)
		require.NoError(t, err)
		assert.Equal(t, expenseAccountID, resolved.Expense)
		assert.Equal(t, payableAccountID, resolved.Payable)
		assert.Equal(t, cashAccountID, resolved.Cash)
	})

	t.Run("missing expense account errors", func(t *testing.T) {
		_, err := ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{},
			&tenant.AccountingControl{DefaultAPAccountID: payableAccountID}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "purchased transportation")
	})

	t.Run("missing AP account errors", func(t *testing.T) {
		_, err := ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{},
			&tenant.AccountingControl{
				DefaultPurchasedTransportationAccountID: expenseAccountID,
			}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accounts payable")
	})

	t.Run("missing cash account errors only when required", func(t *testing.T) {
		control := &tenant.AccountingControl{
			DefaultPurchasedTransportationAccountID: expenseAccountID,
			DefaultAPAccountID:                      payableAccountID,
		}
		_, err := ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{}, control, false)
		require.NoError(t, err)
		_, err = ResolvePostingAccounts(entity, &tenant.CarrierSettlementControl{}, control, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cash account")
	})

	t.Run("expense not required when every line carries an override", func(t *testing.T) {
		override := overrideAccountID
		mapped := &carriersettlement.CarrierSettlement{
			Lines: []*carriersettlement.CarrierSettlementLine{
				{
					EventType:   carriersettlement.CostEventTypeLinehaulCost,
					AmountMinor: 10_000,
					GLAccountID: &override,
				},
			},
		}
		mapped.SyncTotals()
		_, err := ResolvePostingAccounts(mapped, &tenant.CarrierSettlementControl{},
			&tenant.AccountingControl{DefaultAPAccountID: payableAccountID}, false)
		require.NoError(t, err)
	})
}

package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fundedItem(t *testing.T) *factoring.SubmissionItem {
	t.Helper()
	item := &factoring.SubmissionItem{
		InvoiceNumber:        "INV-1",
		FaceMinor:            1000,
		ExpectedAdvanceMinor: 900,
		ExpectedFeeMinor:     30,
		Status:               factoring.SubmissionItemStatusPending,
	}
	require.NoError(t, item.Fund(now, item.ExpectedFunding()))
	return item
}

func line(role factoring.AccountRole, debit, credit int64) factoring.MovementLine {
	return factoring.MovementLine{Role: role, Debit: debit, Credit: credit}
}

func TestPlanFundingMovement(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		treatment factoring.AccountingTreatment
		lines     []factoring.MovementLine
		effect    factoring.InvoiceEffect
		invoice   int64
	}{
		"sale": {
			treatment: factoring.AccountingTreatmentSale,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 900, 0),
				line(factoring.AccountRoleFeeExpense, 30, 0),
				line(factoring.AccountRoleDueFromFactor, 70, 0),
				line(factoring.AccountRoleAR, 0, 1000),
			},
			effect:  factoring.InvoiceEffectSettle,
			invoice: 1000,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 900, 0),
				line(factoring.AccountRoleFeeExpense, 30, 0),
				line(factoring.AccountRoleLiability, 0, 930),
			},
			effect: factoring.InvoiceEffectNone,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoring.PlanFundingMovement(fundedItem(t), tc.treatment)
			require.NoError(t, err)
			assert.Equal(t, factoring.ReserveEntryFunding, got.Kind)
			assert.Equal(t, tc.lines, got.Lines)
			assert.True(t, got.Balanced())
			assert.Equal(t, tc.effect, got.InvoiceEffect)
			assert.Equal(t, tc.invoice, got.InvoiceMinor)
		})
	}
}

func TestPlanReleaseMovementBalancesWithFee(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		treatment factoring.AccountingTreatment
		cash      int64
		lines     []factoring.MovementLine
		fee       int64
	}{
		"sale paid in full": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      70,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 70, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
		},
		"sale with an extra fee": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      60,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 60, 0),
				line(factoring.AccountRoleFeeExpense, 10, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
			fee: 10,
		},
		"sale with a fee rebate": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      80,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 80, 0),
				line(factoring.AccountRoleFeeExpense, 0, 10),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
			fee: -10,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			cash:      70,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 70, 0),
				line(factoring.AccountRoleLiability, 930, 0),
				line(factoring.AccountRoleAR, 0, 1000),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoring.PlanReleaseMovement(fundedItem(t), tc.treatment, tc.cash)
			require.NoError(t, err)
			assert.Equal(t, tc.lines, got.Lines)
			assert.Equal(t, tc.fee, got.FeeMinor)
			assert.True(t, got.Balanced())
		})
	}
}

func TestPlanReleaseMovementSettlesTheInvoiceOnlyWhenBorrowed(t *testing.T) {
	t.Parallel()

	sale, err := factoring.PlanReleaseMovement(fundedItem(t), factoring.AccountingTreatmentSale, 70)
	require.NoError(t, err)
	assert.Equal(t, factoring.InvoiceEffectNone, sale.InvoiceEffect)

	borrowed, err := factoring.PlanReleaseMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSecuredBorrowing,
		70,
	)
	require.NoError(t, err)
	assert.Equal(t, factoring.InvoiceEffectSettle, borrowed.InvoiceEffect)
	assert.Equal(t, int64(1000), borrowed.InvoiceMinor)
}

func TestPlanReleaseMovementOfAnEmptyReserve(t *testing.T) {
	t.Parallel()

	item := &factoring.SubmissionItem{
		FaceMinor: 1000,
		Status:    factoring.SubmissionItemStatusPending,
	}
	require.NoError(t, item.Fund(now, factoring.ItemFunding{AdvanceMinor: 970, FeeMinor: 30}))

	got, err := factoring.PlanReleaseMovement(item, factoring.AccountingTreatmentSale, 0)
	require.NoError(t, err)
	assert.True(t, got.IsEmpty())
	assert.Zero(t, got.Total())
}

func TestPlanChargebackMovement(t *testing.T) {
	t.Parallel()

	repurchase := factoring.ChargebackRepurchaseMinor(fundedItem(t))
	assert.Equal(t, int64(930), repurchase)

	cases := map[string]struct {
		treatment factoring.AccountingTreatment
		cash      int64
		lines     []factoring.MovementLine
		effect    factoring.InvoiceEffect
	}{
		"sale at the repurchase price": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      repurchase,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 0, 930),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
				line(factoring.AccountRoleAR, 1000, 0),
			},
			effect: factoring.InvoiceEffectReopen,
		},
		"sale with a penalty": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      950,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 0, 950),
				line(factoring.AccountRoleFeeExpense, 20, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
				line(factoring.AccountRoleAR, 1000, 0),
			},
			effect: factoring.InvoiceEffectReopen,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			cash:      repurchase,
			lines: []factoring.MovementLine{
				line(factoring.AccountRoleCash, 0, 930),
				line(factoring.AccountRoleLiability, 930, 0),
			},
			effect: factoring.InvoiceEffectNone,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoring.PlanChargebackMovement(fundedItem(t), tc.treatment, tc.cash)
			require.NoError(t, err)
			assert.Equal(t, tc.lines, got.Lines)
			assert.Equal(t, tc.effect, got.InvoiceEffect)
			assert.True(t, got.Balanced())
		})
	}
}

func TestReverseMovementSwapsEveryLine(t *testing.T) {
	t.Parallel()

	original, err := factoring.PlanChargebackMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSale,
		950,
	)
	require.NoError(t, err)

	reversed := factoring.ReverseMovement(&original)

	assert.Equal(t, factoring.ReserveEntryChargebackReversal, reversed.Kind)
	assert.Equal(t, factoring.InvoiceEffectSettle, reversed.InvoiceEffect)
	assert.Equal(t, original.InvoiceMinor, reversed.InvoiceMinor)
	assert.Equal(t, -original.CashMinor, reversed.CashMinor)
	assert.Equal(t, -original.FeeMinor, reversed.FeeMinor)
	assert.Equal(t, -original.ARMinor, reversed.ARMinor)
	require.Len(t, reversed.Lines, len(original.Lines))
	for idx := range original.Lines {
		assert.Equal(t, original.Lines[idx].Role, reversed.Lines[idx].Role)
		assert.Equal(t, original.Lines[idx].Debit, reversed.Lines[idx].Credit)
		assert.Equal(t, original.Lines[idx].Credit, reversed.Lines[idx].Debit)
	}
	assert.True(t, reversed.Balanced())
}

func TestMovementsRefuseAnUnfundedItem(t *testing.T) {
	t.Parallel()

	item := &factoring.SubmissionItem{FaceMinor: 1000, Status: factoring.SubmissionItemStatusPending}

	_, err := factoring.PlanFundingMovement(item, factoring.AccountingTreatmentSale)
	require.Error(t, err)
	_, err = factoring.PlanReleaseMovement(item, factoring.AccountingTreatmentSale, 0)
	require.Error(t, err)
	_, err = factoring.PlanChargebackMovement(item, factoring.AccountingTreatmentSale, 0)
	require.Error(t, err)
}

func TestMovementsRefuseNegativeCash(t *testing.T) {
	t.Parallel()

	_, err := factoring.PlanReleaseMovement(fundedItem(t), factoring.AccountingTreatmentSale, -1)
	require.Error(t, err)
	_, err = factoring.PlanChargebackMovement(fundedItem(t), factoring.AccountingTreatmentSale, -1)
	require.Error(t, err)
}

func TestReserveEntryRebuildsTheMovement(t *testing.T) {
	t.Parallel()

	for _, treatment := range []factoring.AccountingTreatment{
		factoring.AccountingTreatmentSale,
		factoring.AccountingTreatmentSecuredBorrowing,
	} {
		item := fundedItem(t)
		movement, err := factoring.PlanChargebackMovement(item, treatment, 950)
		require.NoError(t, err)

		entry := factoring.NewReserveEntry(
			item,
			&factoring.Submission{CurrencyCode: "USD"},
			&movement,
			now,
		)
		rebuilt := entry.Movement()

		assert.Equal(t, movement.Lines, rebuilt.Lines, string(treatment))
		assert.Equal(t, movement.InvoiceEffect, rebuilt.InvoiceEffect, string(treatment))
		assert.Equal(t, movement.InvoiceMinor, rebuilt.InvoiceMinor, string(treatment))
	}
}

func TestMovementsRefuseAnUnknownTreatment(t *testing.T) {
	t.Parallel()

	_, err := factoring.PlanFundingMovement(fundedItem(t), factoring.AccountingTreatment(""))
	require.Error(t, err)
}

package factoringledger_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

func fundedItem(t *testing.T) *factoringsubmission.SubmissionItem {
	t.Helper()
	item := &factoringsubmission.SubmissionItem{
		InvoiceNumber:        "INV-1",
		FaceMinor:            1000,
		ExpectedAdvanceMinor: 900,
		ExpectedFeeMinor:     30,
		Status:               factoringsubmission.SubmissionItemStatusPending,
	}
	require.NoError(t, item.Fund(now, item.ExpectedFunding()))
	return item
}

func line(role factoring.AccountRole, debit, credit int64) factoringledger.MovementLine {
	return factoringledger.MovementLine{Role: role, Debit: debit, Credit: credit}
}

func TestPlanFundingMovement(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		treatment factoring.AccountingTreatment
		lines     []factoringledger.MovementLine
		effect    factoringledger.InvoiceEffect
		invoice   int64
	}{
		"sale": {
			treatment: factoring.AccountingTreatmentSale,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 900, 0),
				line(factoring.AccountRoleFeeExpense, 30, 0),
				line(factoring.AccountRoleDueFromFactor, 70, 0),
				line(factoring.AccountRoleAR, 0, 1000),
			},
			effect:  factoringledger.InvoiceEffectSettle,
			invoice: 1000,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 900, 0),
				line(factoring.AccountRoleFeeExpense, 30, 0),
				line(factoring.AccountRoleLiability, 0, 930),
			},
			effect: factoringledger.InvoiceEffectNone,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoringledger.PlanFundingMovement(fundedItem(t), tc.treatment)
			require.NoError(t, err)
			assert.Equal(t, factoringledger.ReserveEntryFunding, got.Kind)
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
		lines     []factoringledger.MovementLine
		fee       int64
	}{
		"sale paid in full": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      70,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 70, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
		},
		"sale with an extra fee": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      60,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 60, 0),
				line(factoring.AccountRoleFeeExpense, 10, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
			fee: 10,
		},
		"sale with a fee rebate": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      80,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 80, 0),
				line(factoring.AccountRoleFeeExpense, 0, 10),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
			},
			fee: -10,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			cash:      70,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 70, 0),
				line(factoring.AccountRoleLiability, 930, 0),
				line(factoring.AccountRoleAR, 0, 1000),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoringledger.PlanReleaseMovement(fundedItem(t), tc.treatment, tc.cash)
			require.NoError(t, err)
			assert.Equal(t, tc.lines, got.Lines)
			assert.Equal(t, tc.fee, got.FeeMinor)
			assert.True(t, got.Balanced())
		})
	}
}

func TestPlanReleaseMovementSettlesTheInvoiceOnlyWhenBorrowed(t *testing.T) {
	t.Parallel()

	sale, err := factoringledger.PlanReleaseMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSale,
		70,
	)
	require.NoError(t, err)
	assert.Equal(t, factoringledger.InvoiceEffectNone, sale.InvoiceEffect)

	borrowed, err := factoringledger.PlanReleaseMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSecuredBorrowing,
		70,
	)
	require.NoError(t, err)
	assert.Equal(t, factoringledger.InvoiceEffectSettle, borrowed.InvoiceEffect)
	assert.Equal(t, int64(1000), borrowed.InvoiceMinor)
}

func TestPlanReleaseMovementOfAnEmptyReserve(t *testing.T) {
	t.Parallel()

	item := &factoringsubmission.SubmissionItem{
		FaceMinor: 1000,
		Status:    factoringsubmission.SubmissionItemStatusPending,
	}
	require.NoError(
		t,
		item.Fund(now, factoringsubmission.ItemFunding{AdvanceMinor: 970, FeeMinor: 30}),
	)

	got, err := factoringledger.PlanReleaseMovement(item, factoring.AccountingTreatmentSale, 0)
	require.NoError(t, err)
	assert.True(t, got.IsEmpty())
	assert.Zero(t, got.Total())
}

func TestPlanChargebackMovement(t *testing.T) {
	t.Parallel()

	repurchase := factoringledger.ChargebackRepurchaseMinor(fundedItem(t))
	assert.Equal(t, int64(930), repurchase)

	cases := map[string]struct {
		treatment factoring.AccountingTreatment
		cash      int64
		lines     []factoringledger.MovementLine
		effect    factoringledger.InvoiceEffect
	}{
		"sale at the repurchase price": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      repurchase,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 0, 930),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
				line(factoring.AccountRoleAR, 1000, 0),
			},
			effect: factoringledger.InvoiceEffectReopen,
		},
		"sale with a penalty": {
			treatment: factoring.AccountingTreatmentSale,
			cash:      950,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 0, 950),
				line(factoring.AccountRoleFeeExpense, 20, 0),
				line(factoring.AccountRoleDueFromFactor, 0, 70),
				line(factoring.AccountRoleAR, 1000, 0),
			},
			effect: factoringledger.InvoiceEffectReopen,
		},
		"secured borrowing": {
			treatment: factoring.AccountingTreatmentSecuredBorrowing,
			cash:      repurchase,
			lines: []factoringledger.MovementLine{
				line(factoring.AccountRoleCash, 0, 930),
				line(factoring.AccountRoleLiability, 930, 0),
			},
			effect: factoringledger.InvoiceEffectNone,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := factoringledger.PlanChargebackMovement(fundedItem(t), tc.treatment, tc.cash)
			require.NoError(t, err)
			assert.Equal(t, tc.lines, got.Lines)
			assert.Equal(t, tc.effect, got.InvoiceEffect)
			assert.True(t, got.Balanced())
		})
	}
}

func TestReverseMovementSwapsEveryLine(t *testing.T) {
	t.Parallel()

	original, err := factoringledger.PlanChargebackMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSale,
		950,
	)
	require.NoError(t, err)

	reversed := factoringledger.ReverseMovement(&original)

	assert.Equal(t, factoringledger.ReserveEntryChargebackReversal, reversed.Kind)
	assert.Equal(t, factoringledger.InvoiceEffectSettle, reversed.InvoiceEffect)
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

	item := &factoringsubmission.SubmissionItem{
		FaceMinor: 1000,
		Status:    factoringsubmission.SubmissionItemStatusPending,
	}

	_, err := factoringledger.PlanFundingMovement(item, factoring.AccountingTreatmentSale)
	require.Error(t, err)
	_, err = factoringledger.PlanReleaseMovement(item, factoring.AccountingTreatmentSale, 0)
	require.Error(t, err)
	_, err = factoringledger.PlanChargebackMovement(item, factoring.AccountingTreatmentSale, 0)
	require.Error(t, err)
}

func TestMovementsRefuseNegativeCash(t *testing.T) {
	t.Parallel()

	_, err := factoringledger.PlanReleaseMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSale,
		-1,
	)
	require.Error(t, err)
	_, err = factoringledger.PlanChargebackMovement(
		fundedItem(t),
		factoring.AccountingTreatmentSale,
		-1,
	)
	require.Error(t, err)
}

func TestReserveEntryRebuildsTheMovement(t *testing.T) {
	t.Parallel()

	for _, treatment := range []factoring.AccountingTreatment{
		factoring.AccountingTreatmentSale,
		factoring.AccountingTreatmentSecuredBorrowing,
	} {
		item := fundedItem(t)
		movement, err := factoringledger.PlanChargebackMovement(item, treatment, 950)
		require.NoError(t, err)

		entry := factoringledger.NewReserveEntry(
			item,
			&factoringsubmission.Submission{CurrencyCode: "USD"},
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

	_, err := factoringledger.PlanFundingMovement(fundedItem(t), factoring.AccountingTreatment(""))
	require.Error(t, err)
}

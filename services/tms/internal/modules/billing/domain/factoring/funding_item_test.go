package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissionItemFund(t *testing.T) {
	t.Parallel()

	item := &factoring.SubmissionItem{
		InvoiceNumber: "INV-1",
		FaceMinor:     1000,
		Status:        factoring.SubmissionItemStatusPending,
	}

	require.Error(t, item.Fund(now, factoring.ItemFunding{AdvanceMinor: -1}))
	require.Error(t, item.Fund(now, factoring.ItemFunding{AdvanceMinor: 990, FeeMinor: 20}))
	assert.Equal(t, factoring.SubmissionItemStatusPending, item.Status)

	require.NoError(t, item.Fund(now, factoring.ItemFunding{AdvanceMinor: 880, FeeMinor: 25}))
	assert.Equal(t, factoring.SubmissionItemStatusFunded, item.Status)
	require.NotNil(t, item.FundedOn)
	assert.Equal(t, now, *item.FundedOn)
	advance, fee, reserve := item.Actuals()
	assert.Equal(t, int64(880), advance)
	assert.Equal(t, int64(25), fee)
	assert.Equal(t, int64(95), reserve)

	require.Error(t, item.Fund(now, factoring.ItemFunding{}))
	require.Error(t, item.Reject("late"))
}

func TestSubmissionItemReject(t *testing.T) {
	t.Parallel()

	item := &factoring.SubmissionItem{InvoiceNumber: "INV-1", Status: factoring.SubmissionItemStatusPending}

	require.Error(t, item.Reject("   "))
	require.NoError(t, item.Reject(" Debtor over its credit limit "))
	assert.Equal(t, factoring.SubmissionItemStatusRejected, item.Status)
	assert.Equal(t, "Debtor over its credit limit", item.RejectionReason)
	assert.False(t, item.Status.IsLive())
}

func TestSubmissionItemSettleAndChargeBack(t *testing.T) {
	t.Parallel()

	settled := fundedItem(t)
	require.NoError(t, settled.Settle(now+10))
	assert.Equal(t, factoring.SubmissionItemStatusSettled, settled.Status)
	require.Error(t, settled.Settle(now+20))
	require.Error(t, settled.ChargeBack(now+20, "unpaid"))

	charged := fundedItem(t)
	require.Error(t, charged.ChargeBack(now+10, ""))
	require.NoError(t, charged.ChargeBack(now+10, "Customer never paid"))
	assert.Equal(t, factoring.SubmissionItemStatusChargedBack, charged.Status)
	assert.False(t, charged.Status.IsLive())
	require.NotNil(t, charged.ChargedBackOn)

	require.NoError(t, charged.ReverseChargeback())
	assert.Equal(t, factoring.SubmissionItemStatusFunded, charged.Status)
	assert.Nil(t, charged.ChargedBackOn)
	assert.Empty(t, charged.ChargebackReason)
	assert.True(t, charged.Status.IsLive())
	require.Error(t, charged.ReverseChargeback())
}

func TestSubmissionResolve(t *testing.T) {
	t.Parallel()

	build := func(statuses ...factoring.SubmissionItemStatus) *factoring.Submission {
		sub := &factoring.Submission{Status: factoring.SubmissionStatusSent}
		for _, status := range statuses {
			sub.Items = append(sub.Items, &factoring.SubmissionItem{
				ID:     pulid.MustNew("facsit_"),
				Status: status,
			})
		}
		return sub
	}

	cases := map[string]struct {
		statuses []factoring.SubmissionItemStatus
		want     factoring.SubmissionStatus
		funded   bool
	}{
		"nothing decided": {
			statuses: []factoring.SubmissionItemStatus{factoring.SubmissionItemStatusPending},
			want:     factoring.SubmissionStatusSent,
		},
		"some funded": {
			statuses: []factoring.SubmissionItemStatus{
				factoring.SubmissionItemStatusFunded,
				factoring.SubmissionItemStatusPending,
			},
			want:   factoring.SubmissionStatusPartially,
			funded: true,
		},
		"all decided": {
			statuses: []factoring.SubmissionItemStatus{
				factoring.SubmissionItemStatusFunded,
				factoring.SubmissionItemStatusRejected,
			},
			want:   factoring.SubmissionStatusFunded,
			funded: true,
		},
		"all rejected": {
			statuses: []factoring.SubmissionItemStatus{
				factoring.SubmissionItemStatusRejected,
				factoring.SubmissionItemStatusRejected,
			},
			want: factoring.SubmissionStatusRejected,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sub := build(tc.statuses...)
			sub.Resolve(now)
			assert.Equal(t, tc.want, sub.Status)
			assert.Equal(t, tc.funded, sub.FundedAt != nil)
		})
	}

	sub := build(factoring.SubmissionItemStatusFunded)
	first := now - 50
	sub.FundedAt = &first
	sub.Status = factoring.SubmissionStatusPartially
	sub.Resolve(now)
	assert.Equal(t, first, *sub.FundedAt)

	cancelled := build(factoring.SubmissionItemStatusFunded)
	cancelled.Status = factoring.SubmissionStatusCancelled
	cancelled.Resolve(now)
	assert.Equal(t, factoring.SubmissionStatusCancelled, cancelled.Status)
	assert.False(t, cancelled.CanFund())
}

func TestEvaluateReadinessFundedAndChargedBack(t *testing.T) {
	t.Parallel()

	funded := readyInputs()
	funded.Invoice.OpenBalanceMinor = 0
	funded.SubmissionNumber = "FS2610000001"
	funded.SubmissionItemStatus = factoring.SubmissionItemStatusFunded
	got := factoring.EvaluateReadiness(funded)
	assert.Equal(t, factoring.VerdictFunded, got.Verdict)
	assert.Zero(t, got.FailedCount())

	charged := readyInputs()
	charged.ChargedBack = true
	got = factoring.EvaluateReadiness(charged)
	assert.Equal(t, factoring.VerdictExcluded, got.Verdict)
	assert.Equal(t, []factoring.CheckCode{factoring.CheckChargedBack}, codes(got.Checks))

	resubmitted := readyInputs()
	resubmitted.ChargedBack = true
	resubmitted.SubmissionNumber = "FS2610000002"
	resubmitted.SubmissionItemStatus = factoring.SubmissionItemStatusPending
	got = factoring.EvaluateReadiness(resubmitted)
	assert.Equal(t, factoring.VerdictSubmitted, got.Verdict)
}

func TestAgreementPostingAccounts(t *testing.T) {
	t.Parallel()

	defaultAR := pulid.MustNew("gla_")
	defaultCash := pulid.MustNew("gla_")
	agreement := &factoring.Agreement{}

	accounts := agreement.PostingAccounts(defaultAR, defaultCash)
	assert.Equal(t, defaultCash, accounts.Cash)
	fields := func(missing []factoring.MissingAccount) []string {
		out := make([]string, 0, len(missing))
		for _, m := range missing {
			out = append(out, m.Field)
		}
		return out
	}
	assert.Equal(
		t,
		[]string{"feeExpenseAccountId", "dueFromFactorAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSale)),
	)
	assert.Equal(
		t,
		[]string{"feeExpenseAccountId", "liabilityAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSecuredBorrowing)),
	)

	agreement.CashAccountID = pulid.MustNew("gla_")
	agreement.FeeExpenseAccountID = pulid.MustNew("gla_")
	agreement.DueFromFactorAccountID = pulid.MustNew("gla_")
	accounts = agreement.PostingAccounts(pulid.Nil, pulid.Nil)
	assert.Equal(t, agreement.CashAccountID, accounts.Cash)
	assert.Equal(
		t,
		[]string{"defaultArAccountId"},
		fields(accounts.Missing(factoring.AccountingTreatmentSale)),
	)

	agreement.ChargebackWindowDays = 90
	assert.Equal(t, now+90*86_400, agreement.ChargebackDeadline(now))
}

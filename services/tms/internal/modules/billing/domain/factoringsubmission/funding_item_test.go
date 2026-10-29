package factoringsubmission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmissionItemFund(t *testing.T) {
	t.Parallel()

	item := &factoringsubmission.SubmissionItem{
		InvoiceNumber: "INV-1",
		FaceMinor:     1000,
		Status:        factoringsubmission.SubmissionItemStatusPending,
	}

	require.Error(t, item.Fund(now, factoringsubmission.ItemFunding{AdvanceMinor: -1}))
	require.Error(
		t,
		item.Fund(now, factoringsubmission.ItemFunding{AdvanceMinor: 990, FeeMinor: 20}),
	)
	assert.Equal(t, factoringsubmission.SubmissionItemStatusPending, item.Status)

	require.NoError(
		t,
		item.Fund(now, factoringsubmission.ItemFunding{AdvanceMinor: 880, FeeMinor: 25}),
	)
	assert.Equal(t, factoringsubmission.SubmissionItemStatusFunded, item.Status)
	require.NotNil(t, item.FundedOn)
	assert.Equal(t, now, *item.FundedOn)
	advance, fee, reserve := item.Actuals()
	assert.Equal(t, int64(880), advance)
	assert.Equal(t, int64(25), fee)
	assert.Equal(t, int64(95), reserve)

	require.Error(t, item.Fund(now, factoringsubmission.ItemFunding{}))
	require.Error(t, item.Reject("late"))
}

func TestSubmissionItemReject(t *testing.T) {
	t.Parallel()

	item := &factoringsubmission.SubmissionItem{
		InvoiceNumber: "INV-1",
		Status:        factoringsubmission.SubmissionItemStatusPending,
	}

	require.Error(t, item.Reject("   "))
	require.NoError(t, item.Reject(" Debtor over its credit limit "))
	assert.Equal(t, factoringsubmission.SubmissionItemStatusRejected, item.Status)
	assert.Equal(t, "Debtor over its credit limit", item.RejectionReason)
	assert.False(t, item.Status.IsLive())
}

func TestSubmissionItemSettleAndChargeBack(t *testing.T) {
	t.Parallel()

	settled := fundedItem(t)
	require.NoError(t, settled.Settle(now+10))
	assert.Equal(t, factoringsubmission.SubmissionItemStatusSettled, settled.Status)
	require.Error(t, settled.Settle(now+20))
	require.Error(t, settled.ChargeBack(now+20, "unpaid"))

	charged := fundedItem(t)
	require.Error(t, charged.ChargeBack(now+10, ""))
	require.NoError(t, charged.ChargeBack(now+10, "Customer never paid"))
	assert.Equal(t, factoringsubmission.SubmissionItemStatusChargedBack, charged.Status)
	assert.False(t, charged.Status.IsLive())
	require.NotNil(t, charged.ChargedBackOn)

	require.NoError(t, charged.ReverseChargeback())
	assert.Equal(t, factoringsubmission.SubmissionItemStatusFunded, charged.Status)
	assert.Nil(t, charged.ChargedBackOn)
	assert.Empty(t, charged.ChargebackReason)
	assert.True(t, charged.Status.IsLive())
	require.Error(t, charged.ReverseChargeback())
}

func TestSubmissionResolve(t *testing.T) {
	t.Parallel()

	build := func(statuses ...factoringsubmission.SubmissionItemStatus) *factoringsubmission.Submission {
		sub := &factoringsubmission.Submission{Status: factoringsubmission.SubmissionStatusSent}
		for _, status := range statuses {
			sub.Items = append(sub.Items, &factoringsubmission.SubmissionItem{
				ID:     pulid.MustNew("facsit_"),
				Status: status,
			})
		}
		return sub
	}

	cases := map[string]struct {
		statuses []factoringsubmission.SubmissionItemStatus
		want     factoringsubmission.SubmissionStatus
		funded   bool
	}{
		"nothing decided": {
			statuses: []factoringsubmission.SubmissionItemStatus{
				factoringsubmission.SubmissionItemStatusPending,
			},
			want: factoringsubmission.SubmissionStatusSent,
		},
		"some funded": {
			statuses: []factoringsubmission.SubmissionItemStatus{
				factoringsubmission.SubmissionItemStatusFunded,
				factoringsubmission.SubmissionItemStatusPending,
			},
			want:   factoringsubmission.SubmissionStatusPartially,
			funded: true,
		},
		"all decided": {
			statuses: []factoringsubmission.SubmissionItemStatus{
				factoringsubmission.SubmissionItemStatusFunded,
				factoringsubmission.SubmissionItemStatusRejected,
			},
			want:   factoringsubmission.SubmissionStatusFunded,
			funded: true,
		},
		"all rejected": {
			statuses: []factoringsubmission.SubmissionItemStatus{
				factoringsubmission.SubmissionItemStatusRejected,
				factoringsubmission.SubmissionItemStatusRejected,
			},
			want: factoringsubmission.SubmissionStatusRejected,
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

	sub := build(factoringsubmission.SubmissionItemStatusFunded)
	first := now - 50
	sub.FundedAt = &first
	sub.Status = factoringsubmission.SubmissionStatusPartially
	sub.Resolve(now)
	assert.Equal(t, first, *sub.FundedAt)

	cancelled := build(factoringsubmission.SubmissionItemStatusFunded)
	cancelled.Status = factoringsubmission.SubmissionStatusCancelled
	cancelled.Resolve(now)
	assert.Equal(t, factoringsubmission.SubmissionStatusCancelled, cancelled.Status)
	assert.False(t, cancelled.CanFund())
}

func TestEvaluateReadinessFundedAndChargedBack(t *testing.T) {
	t.Parallel()

	funded := readyInputs()
	funded.Invoice.OpenBalanceMinor = 0
	funded.SubmissionNumber = "FS2610000001"
	funded.SubmissionItemStatus = factoringsubmission.SubmissionItemStatusFunded
	got := factoringreadiness.EvaluateReadiness(funded)
	assert.Equal(t, factoringreadiness.VerdictFunded, got.Verdict)
	assert.Zero(t, got.FailedCount())

	charged := readyInputs()
	charged.ChargedBack = true
	got = factoringreadiness.EvaluateReadiness(charged)
	assert.Equal(t, factoringreadiness.VerdictExcluded, got.Verdict)
	assert.Equal(
		t,
		[]factoringreadiness.CheckCode{factoringreadiness.CheckChargedBack},
		codes(got.Checks),
	)

	resubmitted := readyInputs()
	resubmitted.ChargedBack = true
	resubmitted.SubmissionNumber = "FS2610000002"
	resubmitted.SubmissionItemStatus = factoringsubmission.SubmissionItemStatusPending
	got = factoringreadiness.EvaluateReadiness(resubmitted)
	assert.Equal(t, factoringreadiness.VerdictSubmitted, got.Verdict)
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

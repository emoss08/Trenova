package factoring_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

var (
	factorID = pulid.MustNew("fac_")
	podType  = pulid.MustNew("dt_")
	rateType = pulid.MustNew("dt_")
	shipA    = pulid.MustNew("shp_")
	shipB    = pulid.MustNew("shp_")
)

func readyInputs() *factoring.ReadinessInputs {
	ack := now - 100
	return &factoring.ReadinessInputs{
		At: now,
		Invoice: factoring.InvoiceFacts{
			Posted:             true,
			OpenBalanceMinor:   125_000,
			FactoringCompanyID: factorID,
		},
		Agreement: &factoring.Agreement{
			Status:        factoring.AgreementStatusActive,
			EffectiveFrom: now - 1000,
		},
		Noa: &factoring.Noa{
			Status:         factoring.NoaStatusAcknowledged,
			AcknowledgedAt: &ack,
		},
		Shipments: []factoring.ShipmentFacts{{
			ShipmentID:      shipA,
			ProNumber:       "PRO-1",
			Available:       true,
			AcceptedTypeIDs: map[pulid.ID]struct{}{rateType: {}},
		}},
		AgreementDocuments: []factoring.DocumentNeed{{TypeID: rateType, Name: "Rate confirmation"}},
	}
}

func codes(checks []factoring.Check) []factoring.CheckCode {
	out := make([]factoring.CheckCode, 0, len(checks))
	for _, c := range checks {
		out = append(out, c.Code)
	}
	return out
}

func TestEvaluateReadinessReady(t *testing.T) {
	t.Parallel()

	got := factoring.EvaluateReadiness(readyInputs())

	assert.Equal(t, factoring.VerdictReady, got.Verdict)
	assert.Zero(t, got.FailedCount())
	assert.Equal(t, []factoring.CheckCode{
		factoring.CheckEligible,
		factoring.CheckFactored,
		factoring.CheckAgreement,
		factoring.CheckNoa,
		factoring.CheckDispute,
		factoring.CheckBillingDocument,
		factoring.CheckAgreementDocument,
	}, codes(got.Checks))
}

func TestEvaluateReadinessExcluded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(*factoring.InvoiceFacts)
		code   factoring.CheckCode
	}{
		"credit memo": {func(f *factoring.InvoiceFacts) { f.CreditMemo = true }, factoring.CheckEligible},
		"voided":      {func(f *factoring.InvoiceFacts) { f.Voided = true }, factoring.CheckEligible},
		"draft":       {func(f *factoring.InvoiceFacts) { f.Posted = false }, factoring.CheckEligible},
		"paid":        {func(f *factoring.InvoiceFacts) { f.OpenBalanceMinor = 0 }, factoring.CheckEligible},
		"override":    {func(f *factoring.InvoiceFacts) { f.Excluded = true }, factoring.CheckEligible},
		"no factor":   {func(f *factoring.InvoiceFacts) { f.FactoringCompanyID = "" }, factoring.CheckFactored},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			tc.mutate(&in.Invoice)

			got := factoring.EvaluateReadiness(in)

			assert.Equal(t, factoring.VerdictExcluded, got.Verdict)
			require.Len(t, got.Checks, 1)
			assert.Equal(t, tc.code, got.Checks[0].Code)
			assert.False(t, got.Checks[0].Passed)
		})
	}
}

func TestEvaluateReadinessAgreement(t *testing.T) {
	t.Parallel()

	ended := now - 1
	cases := map[string]*factoring.Agreement{
		"missing": nil,
		"draft":   {Status: factoring.AgreementStatusDraft, EffectiveFrom: now - 10},
		"future":  {Status: factoring.AgreementStatusActive, EffectiveFrom: now + 10},
		"ended":   {Status: factoring.AgreementStatusActive, EffectiveFrom: now - 10, EffectiveTo: &ended},
	}
	for name, agreement := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			in.Agreement = agreement

			got := factoring.EvaluateReadiness(in)

			assert.Equal(t, factoring.VerdictNotReady, got.Verdict)
			failed := got.Failed()
			require.Len(t, failed, 1)
			assert.Equal(t, factoring.CheckAgreement, failed[0].Code)
			assert.Equal(t, factoring.FixSetUpAgreement, failed[0].Fix)
		})
	}
}

func TestEvaluateReadinessNoa(t *testing.T) {
	t.Parallel()

	sent := now - 50
	ack := now - 500
	expired := now - 10
	cases := map[string]struct {
		noa *factoring.Noa
		fix factoring.FixAction
	}{
		"missing":  {nil, factoring.FixSendNoa},
		"released": {&factoring.Noa{Status: factoring.NoaStatusReleased}, factoring.FixSendNoa},
		"draft":    {&factoring.Noa{Status: factoring.NoaStatusDraft}, factoring.FixSendNoa},
		"sent": {
			&factoring.Noa{Status: factoring.NoaStatusSent, SentAt: &sent},
			factoring.FixRecordNoaAcknowledgement,
		},
		"expired": {
			&factoring.Noa{
				Status:         factoring.NoaStatusAcknowledged,
				AcknowledgedAt: &ack,
				ExpiresAt:      &expired,
			},
			factoring.FixSendNoa,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			in.Noa = tc.noa

			got := factoring.EvaluateReadiness(in)

			failed := got.Failed()
			require.Len(t, failed, 1)
			assert.Equal(t, factoring.CheckNoa, failed[0].Code)
			assert.Equal(t, tc.fix, failed[0].Fix)
		})
	}
}

func TestEvaluateReadinessDispute(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.OpenDispute = true

	got := factoring.EvaluateReadiness(in)

	failed := got.Failed()
	require.Len(t, failed, 1)
	assert.Equal(t, factoring.CheckDispute, failed[0].Code)
	assert.Equal(t, factoring.FixResolveDispute, failed[0].Fix)
}

func TestEvaluateReadinessShipmentFailures(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.AgreementDocuments = nil
	in.Shipments = []factoring.ShipmentFacts{
		{
			ShipmentID:       shipA,
			ProNumber:        "PRO-1",
			Available:        true,
			MissingDocuments: []factoring.DocumentNeed{{TypeID: podType, Name: "Proof of delivery"}},
			ValidationFailures: []factoring.ShipmentValidation{
				{Code: "RATE_MISSING", Message: "The shipment has no rate"},
			},
		},
		{ShipmentID: shipB, ProNumber: "PRO-2"},
	}

	got := factoring.EvaluateReadiness(in)

	assert.Equal(t, factoring.VerdictNotReady, got.Verdict)
	failed := got.Failed()
	require.Len(t, failed, 3)
	assert.Equal(t, factoring.CheckBillingDocument, failed[0].Code)
	assert.Equal(t, podType, failed[0].DocumentTypeID)
	assert.Equal(t, shipA, failed[0].ShipmentID)
	assert.Equal(t, factoring.FixRequestDocument, failed[0].Fix)
	assert.Equal(t, factoring.CheckBillingValidation, failed[1].Code)
	assert.Equal(t, "RATE_MISSING", failed[1].Detail)
	assert.Equal(t, factoring.CheckShipmentUnavailable, failed[2].Code)
	assert.Equal(t, shipB, failed[2].ShipmentID)
}

func TestEvaluateReadinessAgreementDocuments(t *testing.T) {
	t.Parallel()

	t.Run("missing on one leg", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments = append(in.Shipments, factoring.ShipmentFacts{
			ShipmentID: shipB,
			Available:  true,
		})

		got := factoring.EvaluateReadiness(in)

		failed := got.Failed()
		require.Len(t, failed, 1)
		assert.Equal(t, factoring.CheckAgreementDocument, failed[0].Code)
		assert.True(t, failed[0].ShipmentID.IsNil(), "a multi-leg need is not pinned to one leg")
	})

	t.Run("attached to the invoice", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments[0].AcceptedTypeIDs = nil
		in.InvoiceAcceptedTypeIDs = map[pulid.ID]struct{}{rateType: {}}

		got := factoring.EvaluateReadiness(in)

		assert.Equal(t, factoring.VerdictReady, got.Verdict)
	})

	t.Run("missing on a single leg", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments[0].AcceptedTypeIDs = nil

		got := factoring.EvaluateReadiness(in)

		failed := got.Failed()
		require.Len(t, failed, 1)
		assert.Equal(t, shipA, failed[0].ShipmentID)
		assert.Equal(t, factoring.FixRequestDocument, failed[0].Fix)
	})
}

func TestNextState(t *testing.T) {
	t.Parallel()

	ready := &factoring.Readiness{Verdict: factoring.VerdictReady}
	notReady := &factoring.Readiness{
		Verdict: factoring.VerdictNotReady,
		Checks:  []factoring.Check{{Code: factoring.CheckNoa}},
	}

	first := &factoring.InvoiceState{}
	tr := factoring.NextState(nil, first, ready, now)
	assert.True(t, tr.BecameReady)
	require.NotNil(t, first.ReadySince)
	assert.Equal(t, now, *first.ReadySince)

	announced := now + 5
	first.ReadyAnnouncedAt = &announced
	second := &factoring.InvoiceState{}
	tr = factoring.NextState(first, second, ready, now+60)
	assert.False(t, tr.BecameReady)
	assert.Equal(t, now, *second.ReadySince, "ready since holds across evaluations")
	assert.Equal(t, &announced, second.ReadyAnnouncedAt)
	assert.True(t, second.ReadyLongerThan(now+factoring.ReadyUnsubmittedSeconds, factoring.ReadyUnsubmittedSeconds))

	third := &factoring.InvoiceState{}
	tr = factoring.NextState(second, third, notReady, now+120)
	assert.True(t, tr.LeftReady)
	assert.Nil(t, third.ReadySince)
	assert.Nil(t, third.ReadyAnnouncedAt)
	assert.Equal(t, 1, third.FailedChecks)

	fourth := &factoring.InvoiceState{}
	tr = factoring.NextState(third, fourth, ready, now+180)
	assert.True(t, tr.BecameReady, "a new ready period is announced again")
	assert.Nil(t, fourth.ReadyAnnouncedAt)
}

func TestEvaluateReadinessSubmitted(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.SubmissionNumber = "FS2610000001"
	in.OpenDispute = true
	got := factoring.EvaluateReadiness(in)
	assert.Equal(t, factoring.VerdictSubmitted, got.Verdict)
	assert.Equal(t, 1, got.FailedCount())

	in.Invoice.OpenBalanceMinor = 0
	assert.Equal(t, factoring.VerdictExcluded, factoring.EvaluateReadiness(in).Verdict)
}

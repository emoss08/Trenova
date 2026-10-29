package factoringreadiness_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
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

func readyInputs() *factoringreadiness.ReadinessInputs {
	ack := now - 100
	return &factoringreadiness.ReadinessInputs{
		At: now,
		Invoice: factoringreadiness.InvoiceFacts{
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
		Shipments: []factoringreadiness.ShipmentFacts{{
			ShipmentID:      shipA,
			ProNumber:       "PRO-1",
			Available:       true,
			AcceptedTypeIDs: map[pulid.ID]struct{}{rateType: {}},
		}},
		AgreementDocuments: []factoringreadiness.DocumentNeed{
			{TypeID: rateType, Name: "Rate confirmation"},
		},
	}
}

func codes(checks []factoringreadiness.Check) []factoringreadiness.CheckCode {
	out := make([]factoringreadiness.CheckCode, 0, len(checks))
	for _, c := range checks {
		out = append(out, c.Code)
	}
	return out
}

func TestEvaluateReadinessReady(t *testing.T) {
	t.Parallel()

	got := factoringreadiness.EvaluateReadiness(readyInputs())

	assert.Equal(t, factoringreadiness.VerdictReady, got.Verdict)
	assert.Zero(t, got.FailedCount())
	assert.Equal(t, []factoringreadiness.CheckCode{
		factoringreadiness.CheckEligible,
		factoringreadiness.CheckFactored,
		factoringreadiness.CheckAgreement,
		factoringreadiness.CheckNoa,
		factoringreadiness.CheckDispute,
		factoringreadiness.CheckBillingDocument,
		factoringreadiness.CheckAgreementDocument,
	}, codes(got.Checks))
}

func TestEvaluateReadinessExcluded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mutate func(*factoringreadiness.InvoiceFacts)
		code   factoringreadiness.CheckCode
	}{
		"credit memo": {
			func(f *factoringreadiness.InvoiceFacts) { f.CreditMemo = true },
			factoringreadiness.CheckEligible,
		},
		"voided": {
			func(f *factoringreadiness.InvoiceFacts) { f.Voided = true },
			factoringreadiness.CheckEligible,
		},
		"draft": {
			func(f *factoringreadiness.InvoiceFacts) { f.Posted = false },
			factoringreadiness.CheckEligible,
		},
		"paid": {
			func(f *factoringreadiness.InvoiceFacts) { f.OpenBalanceMinor = 0 },
			factoringreadiness.CheckEligible,
		},
		"override": {
			func(f *factoringreadiness.InvoiceFacts) { f.Excluded = true },
			factoringreadiness.CheckEligible,
		},
		"no factor": {
			func(f *factoringreadiness.InvoiceFacts) { f.FactoringCompanyID = "" },
			factoringreadiness.CheckFactored,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			tc.mutate(&in.Invoice)

			got := factoringreadiness.EvaluateReadiness(in)

			assert.Equal(t, factoringreadiness.VerdictExcluded, got.Verdict)
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
		"ended": {
			Status:        factoring.AgreementStatusActive,
			EffectiveFrom: now - 10,
			EffectiveTo:   &ended,
		},
	}
	for name, agreement := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			in.Agreement = agreement

			got := factoringreadiness.EvaluateReadiness(in)

			assert.Equal(t, factoringreadiness.VerdictNotReady, got.Verdict)
			failed := got.Failed()
			require.Len(t, failed, 1)
			assert.Equal(t, factoringreadiness.CheckAgreement, failed[0].Code)
			assert.Equal(t, factoringreadiness.FixSetUpAgreement, failed[0].Fix)
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
		fix factoringreadiness.FixAction
	}{
		"missing": {nil, factoringreadiness.FixSendNoa},
		"released": {
			&factoring.Noa{Status: factoring.NoaStatusReleased},
			factoringreadiness.FixSendNoa,
		},
		"draft": {
			&factoring.Noa{Status: factoring.NoaStatusDraft},
			factoringreadiness.FixSendNoa,
		},
		"sent": {
			&factoring.Noa{Status: factoring.NoaStatusSent, SentAt: &sent},
			factoringreadiness.FixRecordNoaAcknowledgement,
		},
		"expired": {
			&factoring.Noa{
				Status:         factoring.NoaStatusAcknowledged,
				AcknowledgedAt: &ack,
				ExpiresAt:      &expired,
			},
			factoringreadiness.FixSendNoa,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := readyInputs()
			in.Noa = tc.noa

			got := factoringreadiness.EvaluateReadiness(in)

			failed := got.Failed()
			require.Len(t, failed, 1)
			assert.Equal(t, factoringreadiness.CheckNoa, failed[0].Code)
			assert.Equal(t, tc.fix, failed[0].Fix)
		})
	}
}

func TestEvaluateReadinessDispute(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.OpenDispute = true

	got := factoringreadiness.EvaluateReadiness(in)

	failed := got.Failed()
	require.Len(t, failed, 1)
	assert.Equal(t, factoringreadiness.CheckDispute, failed[0].Code)
	assert.Equal(t, factoringreadiness.FixResolveDispute, failed[0].Fix)
}

func TestEvaluateReadinessShipmentFailures(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.AgreementDocuments = nil
	in.Shipments = []factoringreadiness.ShipmentFacts{
		{
			ShipmentID: shipA,
			ProNumber:  "PRO-1",
			Available:  true,
			MissingDocuments: []factoringreadiness.DocumentNeed{
				{TypeID: podType, Name: "Proof of delivery"},
			},
			ValidationFailures: []factoringreadiness.ShipmentValidation{
				{Code: "RATE_MISSING", Message: "The shipment has no rate"},
			},
		},
		{ShipmentID: shipB, ProNumber: "PRO-2"},
	}

	got := factoringreadiness.EvaluateReadiness(in)

	assert.Equal(t, factoringreadiness.VerdictNotReady, got.Verdict)
	failed := got.Failed()
	require.Len(t, failed, 3)
	assert.Equal(t, factoringreadiness.CheckBillingDocument, failed[0].Code)
	assert.Equal(t, podType, failed[0].DocumentTypeID)
	assert.Equal(t, shipA, failed[0].ShipmentID)
	assert.Equal(t, factoringreadiness.FixRequestDocument, failed[0].Fix)
	assert.Equal(t, factoringreadiness.CheckBillingValidation, failed[1].Code)
	assert.Equal(t, "RATE_MISSING", failed[1].Detail)
	assert.Equal(t, factoringreadiness.CheckShipmentUnavailable, failed[2].Code)
	assert.Equal(t, shipB, failed[2].ShipmentID)
}

func TestEvaluateReadinessAgreementDocuments(t *testing.T) {
	t.Parallel()

	t.Run("missing on one leg", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments = append(in.Shipments, factoringreadiness.ShipmentFacts{
			ShipmentID: shipB,
			Available:  true,
		})

		got := factoringreadiness.EvaluateReadiness(in)

		failed := got.Failed()
		require.Len(t, failed, 1)
		assert.Equal(t, factoringreadiness.CheckAgreementDocument, failed[0].Code)
		assert.True(t, failed[0].ShipmentID.IsNil(), "a multi-leg need is not pinned to one leg")
	})

	t.Run("attached to the invoice", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments[0].AcceptedTypeIDs = nil
		in.InvoiceAcceptedTypeIDs = map[pulid.ID]struct{}{rateType: {}}

		got := factoringreadiness.EvaluateReadiness(in)

		assert.Equal(t, factoringreadiness.VerdictReady, got.Verdict)
	})

	t.Run("missing on a single leg", func(t *testing.T) {
		t.Parallel()
		in := readyInputs()
		in.Shipments[0].AcceptedTypeIDs = nil

		got := factoringreadiness.EvaluateReadiness(in)

		failed := got.Failed()
		require.Len(t, failed, 1)
		assert.Equal(t, shipA, failed[0].ShipmentID)
		assert.Equal(t, factoringreadiness.FixRequestDocument, failed[0].Fix)
	})
}

func TestNextState(t *testing.T) {
	t.Parallel()

	ready := &factoringreadiness.Readiness{Verdict: factoringreadiness.VerdictReady}
	notReady := &factoringreadiness.Readiness{
		Verdict: factoringreadiness.VerdictNotReady,
		Checks:  []factoringreadiness.Check{{Code: factoringreadiness.CheckNoa}},
	}

	first := &factoringreadiness.InvoiceState{}
	tr := factoringreadiness.NextState(nil, first, ready, now)
	assert.True(t, tr.BecameReady)
	require.NotNil(t, first.ReadySince)
	assert.Equal(t, now, *first.ReadySince)

	announced := now + 5
	first.ReadyAnnouncedAt = &announced
	second := &factoringreadiness.InvoiceState{}
	tr = factoringreadiness.NextState(first, second, ready, now+60)
	assert.False(t, tr.BecameReady)
	assert.Equal(t, now, *second.ReadySince, "ready since holds across evaluations")
	assert.Equal(t, &announced, second.ReadyAnnouncedAt)
	assert.True(
		t,
		second.ReadyLongerThan(
			now+factoringreadiness.ReadyUnsubmittedSeconds,
			factoringreadiness.ReadyUnsubmittedSeconds,
		),
	)

	third := &factoringreadiness.InvoiceState{}
	tr = factoringreadiness.NextState(second, third, notReady, now+120)
	assert.True(t, tr.LeftReady)
	assert.Nil(t, third.ReadySince)
	assert.Nil(t, third.ReadyAnnouncedAt)
	assert.Equal(t, 1, third.FailedChecks)

	fourth := &factoringreadiness.InvoiceState{}
	tr = factoringreadiness.NextState(third, fourth, ready, now+180)
	assert.True(t, tr.BecameReady, "a new ready period is announced again")
	assert.Nil(t, fourth.ReadyAnnouncedAt)
}

func TestEvaluateReadinessSubmitted(t *testing.T) {
	t.Parallel()

	in := readyInputs()
	in.SubmissionNumber = "FS2610000001"
	in.OpenDispute = true
	got := factoringreadiness.EvaluateReadiness(in)
	assert.Equal(t, factoringreadiness.VerdictSubmitted, got.Verdict)
	assert.Equal(t, 1, got.FailedCount())

	in.Invoice.OpenBalanceMinor = 0
	assert.Equal(
		t,
		factoringreadiness.VerdictExcluded,
		factoringreadiness.EvaluateReadiness(in).Verdict,
	)
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

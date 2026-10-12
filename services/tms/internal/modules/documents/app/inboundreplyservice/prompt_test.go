package inboundreplyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/stretchr/testify/assert"
)

func TestDecisionInstructionFollowsWhatTheWaiverDid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		state inboundmessage.ActionState
		want  string
	}{
		{name: "undecided", state: inboundmessage.ActionState{}, want: ""},
		{
			name:  "stand by",
			state: inboundmessage.ActionState{StopsDecision: inboundmessage.StopsDecisionStandBy},
			want:  standByInstruction,
		},
		{
			name:  "waived before the waiver was recorded",
			state: inboundmessage.ActionState{StopsDecision: inboundmessage.StopsDecisionWaive},
			want:  waiveInstruction,
		},
		{
			name: "waived before invoicing",
			state: inboundmessage.ActionState{
				StopsDecision:   inboundmessage.StopsDecisionWaive,
				DetentionWaiver: &inboundmessage.DetentionWaiver{Amount: "185.00", Currency: "USD"},
			},
			want: waiveUnbilledInstruction,
		},
		{
			name: "credited",
			state: inboundmessage.ActionState{
				StopsDecision: inboundmessage.StopsDecisionWaive,
				DetentionWaiver: &inboundmessage.DetentionWaiver{
					Amount:         "185.00",
					InvoiceNumbers: []string{"INV-20931"},
				},
			},
			want: waiveCreditedInstruction,
		},
		{
			name: "credit waiting for approval",
			state: inboundmessage.ActionState{
				StopsDecision: inboundmessage.StopsDecisionWaive,
				DetentionWaiver: &inboundmessage.DetentionWaiver{
					Amount:         "185.00",
					InvoiceNumbers: []string{"INV-20931"},
					CreditPending:  true,
				},
			},
			want: waivePendingInstruction,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, decisionInstruction(&tc.state))
		})
	}
}

func TestDecisionFactsNameTheInvoiceAndTheCredit(t *testing.T) {
	t.Parallel()

	credited := inboundmessage.ActionState{
		StopsDecision: inboundmessage.StopsDecisionWaive,
		DetentionWaiver: &inboundmessage.DetentionWaiver{
			Amount:            "185.00",
			Currency:          "USD",
			InvoiceNumbers:    []string{"INV-20931"},
			CreditMemoNumbers: []string{"CM-00012"},
		},
	}
	assert.Equal(t,
		"Decision: waive the charge\nAmount waived: 185.00 USD\nInvoice credited: INV-20931\n"+
			"Credit memo: CM-00012\nCredit status: issued",
		decisionFacts(&credited),
	)

	credited.DetentionWaiver.CreditMemoNumbers = nil
	credited.DetentionWaiver.CreditPending = true
	assert.Equal(t,
		"Decision: waive the charge\nAmount waived: 185.00 USD\nInvoice to credit: INV-20931\n"+
			"Credit status: waiting for approval in billing",
		decisionFacts(&credited),
	)

	unbilled := inboundmessage.ActionState{
		StopsDecision:   inboundmessage.StopsDecisionWaive,
		DetentionWaiver: &inboundmessage.DetentionWaiver{Amount: "185.00", Currency: "USD"},
	}
	assert.Equal(t,
		"Decision: waive the charge\nAmount waived: 185.00 USD\nInvoiced: no, it was removed before invoicing",
		decisionFacts(&unbilled),
	)

	assert.Equal(t,
		"Decision: stand by the charge",
		decisionFacts(&inboundmessage.ActionState{StopsDecision: inboundmessage.StopsDecisionStandBy}),
	)
	assert.Empty(t, decisionFacts(&inboundmessage.ActionState{}))
}

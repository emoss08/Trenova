package triageruleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outcomeRow(
	domain string,
	class inboundmessage.Classification,
	outcome triage.ActivityKind,
	count int,
) *triageports.ManualOutcomeRow {
	return &triageports.ManualOutcomeRow{
		SenderDomain:   domain,
		Classification: class,
		Outcome:        outcome,
		Count:          count,
		Arrived:        count + 2,
		Mailboxes:      1,
		MailboxID:      "imbx_intake",
	}
}

func TestBestSuggestion_NeedsFiveOfTheSameOutcome(t *testing.T) {
	t.Parallel()

	rows := []*triageports.ManualOutcomeRow{
		outcomeRow("acme.example", inboundmessage.ClassificationInvoice, triage.ActivityFiled, 4),
	}
	assert.Nil(t, BestSuggestion(rows, nil))

	rows[0].Count = 5
	suggestion := BestSuggestion(rows, nil)
	require.NotNil(t, suggestion)
	assert.Equal(t, inboundmessage.NovaActionFileOnShipment, suggestion.Action)
	assert.Equal(t, triage.SenderDomain, suggestion.SenderScope)
	assert.Equal(t, "acme.example", suggestion.Sender)
	assert.Equal(t, 5, suggestion.HandledByHand)
	assert.Equal(t, 6, suggestion.Total)
	require.NotNil(t, suggestion.MailboxID)
	assert.Equal(t, pulid.ID("imbx_intake"), *suggestion.MailboxID)
}

func TestBestSuggestion_NeedsNinetyPercentAgreement(t *testing.T) {
	t.Parallel()

	rows := []*triageports.ManualOutcomeRow{
		outcomeRow("news.example", inboundmessage.ClassificationOther, triage.ActivityIgnored, 9),
		outcomeRow("news.example", inboundmessage.ClassificationOther, triage.ActivityHandled, 2),
	}
	assert.Nil(t, BestSuggestion(rows, nil), "9 of 11 is under ninety percent")

	rows[0].Count = 18
	suggestion := BestSuggestion(rows, nil)
	require.NotNil(t, suggestion)
	assert.Equal(t, inboundmessage.NovaActionIgnore, suggestion.Action)
}

func TestBestSuggestion_HandlingByHandIsNotARuleAction(t *testing.T) {
	t.Parallel()

	rows := []*triageports.ManualOutcomeRow{
		outcomeRow("acme.example", inboundmessage.ClassificationTender, triage.ActivityHandled, 40),
	}
	assert.Nil(t, BestSuggestion(rows, nil))
}

func TestBestSuggestion_SkipsWhatARuleAlreadyCovers(t *testing.T) {
	t.Parallel()

	rows := []*triageports.ManualOutcomeRow{
		outcomeRow("acme.example", inboundmessage.ClassificationInvoice, triage.ActivityFiled, 12),
		outcomeRow("drivers.example", inboundmessage.ClassificationProofOfDelivery,
			triage.ActivityFiled, 6),
	}
	covering := &triage.Rule{
		ID:             "trule_acme",
		Classification: inboundmessage.ClassificationInvoice,
		SenderScope:    triage.SenderDomain,
		Sender:         "acme.example",
		Action:         inboundmessage.NovaActionIgnore,
		Enabled:        false,
	}

	suggestion := BestSuggestion(rows, []*triage.Rule{covering})

	require.NotNil(t, suggestion)
	assert.Equal(t, "drivers.example", suggestion.Sender)
}

func TestBestSuggestion_PrefersTheMostHandledByHand(t *testing.T) {
	t.Parallel()

	rows := []*triageports.ManualOutcomeRow{
		outcomeRow("small.example", inboundmessage.ClassificationInvoice, triage.ActivityFiled, 6),
		outcomeRow("big.example", inboundmessage.ClassificationInvoice, triage.ActivityFiled, 30),
	}
	rows[1].Mailboxes = 2

	suggestion := BestSuggestion(rows, nil)

	require.NotNil(t, suggestion)
	assert.Equal(t, "big.example", suggestion.Sender)
	assert.Nil(t, suggestion.MailboxID, "mail from several sources gets a rule for every source")
}

func TestBestSuggestion_AnyCustomerRuleCoversCustomerMail(t *testing.T) {
	t.Parallel()

	row := outcomeRow("acme.example", inboundmessage.ClassificationInvoice, triage.ActivityFiled, 8)
	row.AllCustomers = true
	rule := &triage.Rule{
		Classification: inboundmessage.ClassificationInvoice,
		SenderScope:    triage.SenderAnyCustomer,
		Action:         inboundmessage.NovaActionFileOnShipment,
	}

	assert.Nil(t, BestSuggestion([]*triageports.ManualOutcomeRow{row}, []*triage.Rule{rule}))
}

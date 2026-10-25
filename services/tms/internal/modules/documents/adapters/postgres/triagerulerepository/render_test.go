package triagerulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderSentence(t *testing.T, rule *triage.Rule) (string, []any) {
	t.Helper()

	return keeltest.Render(t, sentenceQuery(triageports.FindRuleBySentenceRequest{
		Sentence: rule,
	}).Limit(1))
}

func TestSentenceQueryMatchesTheWholeSentence(t *testing.T) {
	t.Parallel()

	sqlText, args := renderSentence(t, &triage.Rule{
		Classification: inboundmessage.ClassificationRateConfirmation,
		SenderScope:    triage.SenderDomain,
		Sender:         "acme.com",
		Action:         inboundmessage.NovaActionFileOnShipment,
	})

	assert.Contains(
		t,
		sqlText,
		`FROM "triage_rules" tr WHERE tr."business_unit_id" = $1 AND tr."organization_id" = $2 `+
			`AND tr."classification" = 'RateConfirmation' AND tr."sender_scope" = 'Domain' `+
			`AND (coalesce(tr."sender", '') = $3) AND (coalesce(tr."mailbox_id", '') = $4) `+
			`AND tr."action" = 'FileOnShipment' LIMIT 1`,
	)
	assert.Equal(t, []any{
		"bu_1",
		"org_1",
		"acme.com",
		"",
	}, args)
}

func TestSentenceQueryLeavesTheRuleBeingEditedOut(t *testing.T) {
	t.Parallel()

	sqlText, args := renderSentence(t, &triage.Rule{
		ID:             "trl_1",
		Classification: inboundmessage.ClassificationInvoice,
		SenderScope:    triage.SenderAnyCarrier,
		MailboxID:      "imbx_1",
		Action:         inboundmessage.NovaActionIgnore,
	})

	assert.Contains(
		t,
		sqlText,
		`AND tr."classification" = 'Invoice' AND tr."sender_scope" = 'AnyCarrier' `+
			`AND (coalesce(tr."sender", '') = $3) AND (coalesce(tr."mailbox_id", '') = $4) `+
			`AND tr."action" = 'Ignore' AND tr."id" <> $5 LIMIT 1`,
	)
	require.Len(t, args, 5)
	assert.Equal(t, "imbx_1", args[3])
	assert.Equal(t, "trl_1", args[4])
}

package triagerulerepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func renderSentence(rule *triage.Rule) string {
	db := bun.NewDB(new(sql.DB), pgdialect.New())
	rules := make([]*triage.Rule, 0, 1)

	return sentenceQuery(db.NewSelect().Model(&rules), triageports.FindRuleBySentenceRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		Sentence:   rule,
	}).String()
}

func TestSentenceQueryMatchesTheWholeSentence(t *testing.T) {
	t.Parallel()

	sqlText := renderSentence(&triage.Rule{
		Classification: inboundmessage.ClassificationRateConfirmation,
		SenderScope:    triage.SenderDomain,
		Sender:         "acme.com",
		Action:         inboundmessage.NovaActionFileOnShipment,
	})

	assert.Contains(t, sqlText, "organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "business_unit_id = 'bu_1'", sqlText)
	assert.Contains(t, sqlText, "'RateConfirmation'", sqlText)
	assert.Contains(t, sqlText, "'Domain'", sqlText)
	assert.Contains(t, sqlText, "COALESCE(trule.sender, '') = 'acme.com'", sqlText)
	assert.Contains(t, sqlText, "COALESCE(trule.mailbox_id, '') = ''", sqlText)
	assert.Contains(t, sqlText, "'FileOnShipment'", sqlText)
	assert.NotContains(t, sqlText, "trule.id <>", sqlText)
}

func TestSentenceQueryLeavesTheRuleBeingEditedOut(t *testing.T) {
	t.Parallel()

	sqlText := renderSentence(&triage.Rule{
		ID:             "trl_1",
		Classification: inboundmessage.ClassificationInvoice,
		SenderScope:    triage.SenderAnyCarrier,
		MailboxID:      "imbx_1",
		Action:         inboundmessage.NovaActionIgnore,
	})

	assert.Contains(t, sqlText, "COALESCE(trule.mailbox_id, '') = 'imbx_1'", sqlText)
	assert.Contains(t, sqlText, "trule.id <> 'trl_1'", sqlText)
}

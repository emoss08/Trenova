package triagemessagerepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestClaimRuleOnlyTakesAnUnclaimedMessage(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	sqlText := claimRuleQuery(db, &triageports.ClaimRuleRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		MessageID:  "imsg_1",
		RuleID:     "trl_1",
		Action:     inboundmessage.NovaActionIgnore,
		At:         1_760_290_000,
	}).String()

	assert.Contains(t, sqlText, "handled_by_rule_id = 'trl_1'", sqlText)
	assert.Contains(t, sqlText, "organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "business_unit_id = 'bu_1'", sqlText)
	assert.Contains(t, sqlText, "'imsg_1'", sqlText)
	assert.Contains(t, sqlText, "handled_by_rule_id IS NULL", sqlText)
	assert.Contains(t, sqlText, "RETURNING version", sqlText)
}

func TestJSONOrNullStoresAnEmptyListAsNull(t *testing.T) {
	t.Parallel()

	empty, err := jsonOrNull([]inboundmessage.Fact{})
	require.NoError(t, err)
	assert.Nil(t, empty)

	encoded, err := jsonOrNull([]inboundmessage.Fact{{Key: "rate", Value: "$1,800"}})
	require.NoError(t, err)
	require.NotNil(t, encoded)
	assert.Contains(t, *encoded, `"rate"`)
	assert.Contains(t, *encoded, `"$1,800"`)
}

func TestHintConflictTargetsTheSenderClassification(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"CONFLICT (organization_id, business_unit_id, sender_domain, classification) DO UPDATE",
		hintConflict(),
	)
}

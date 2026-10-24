package triagemessagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimRuleOnlyTakesAnUnclaimedMessage(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, claimRuleQuery(&triageports.ClaimRuleRequest{
		MessageID: "imsg_1",
		RuleID:    "trl_1",
		Action:    inboundmessage.NovaActionIgnore,
		At:        1_760_290_000,
	}))

	assert.Equal(
		t,
		`SELECT im."version", im."id", im."business_unit_id", im."organization_id" `+
			`FROM "inbound_messages" im `+
			`WHERE im."business_unit_id" = $1 AND im."organization_id" = $2 `+
			`AND im."id" = $3 AND im."handled_by_rule_id" IS NULL`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "imsg_1"}, args)
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

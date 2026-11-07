package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestRecordReadQueryLooksOnlyAtLaterToolResultsInTheConversation(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, recordReadQuery(&agentwaitports.RecordReadSinceRequest{
		ThreadID:   pulid.ID("athr_1"),
		RecordID:   pulid.ID("bqi_1"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")},
		Since:      1_800_000_000,
	}))
	for _, column := range []string{
		`"organization_id" = `, `"business_unit_id" = `, `"thread_id" = `, `"role" = `,
		`"tool_failed" IS FALSE`, `"created_at" > `, `"content" LIKE `,
	} {
		assert.Contains(t, sql, column)
	}
	assert.Contains(t, args, "org_1")
	assert.Contains(t, args, "athr_1")
	assert.Contains(t, args, `%bqi\_1%`, "the id is matched literally, not as a pattern")
}

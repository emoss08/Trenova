package assistantqueuerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestListQueryOrdersTheQueue(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, listQuery(&agentports.AssistantQueueScope{
		ThreadID: pulid.ID("ath_1"),
		UserID:   pulid.ID("usr_1"),
	}))
	assert.Contains(t, sql, `"position" ASC`)
	assert.Contains(t, args, "usr_1")
}

func TestTallyQueryCountsTheThread(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, tallyQuery(pulid.ID("ath_1")))
	assert.Contains(t, sql, "count(*)")
	assert.Contains(t, sql, "COALESCE(MAX(")
}

func TestHeadQuerySkipsLockedRows(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, headQuery(&agentports.AssistantQueueScope{ThreadID: pulid.ID("ath_1")}))
	assert.Contains(t, sql, "FOR UPDATE")
	assert.Contains(t, sql, "SKIP LOCKED")
	assert.Contains(t, sql, "LIMIT 1")
}

func TestEditQueryComparesTheVersion(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, editQuery(&conversation.QueuedMessage{
		ID:       pulid.ID("aqm_1"),
		ThreadID: pulid.ID("ath_1"),
		UserID:   pulid.ID("usr_1"),
	}, 3))
	assert.Contains(t, sql, `"version" = `)
	assert.Contains(t, args, int64(3))
}

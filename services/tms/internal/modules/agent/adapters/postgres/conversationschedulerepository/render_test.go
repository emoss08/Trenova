package conversationschedulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestListByNextRunPutsTheSchedulesThatFireFirst(t *testing.T) {
	t.Parallel()

	req := &conversationscheduleports.ListConversationSchedulesRequest{
		UserID: pulid.MustNew("usr_"),
		Order:  conversationscheduleports.ConversationScheduleOrderNextRun,
	}
	sql, _ := keeltest.Render(t, listPageQuery(req, 25))

	assert.Contains(
		t,
		sql,
		`ORDER BY cs."enabled" DESC, cs."next_run_at" ASC NULLS LAST, cs."created_at" DESC, cs."id" DESC LIMIT 25`,
		sql,
	)
	assert.NotContains(t, sql, `"thread_id" =`, sql)
}

func TestListReadsTheTitlesOfThePagesConversationsOnly(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	page := listPageQuery(&conversationscheduleports.ListConversationSchedulesRequest{
		UserID:   pulid.MustNew("usr_"),
		ThreadID: threadID,
		Offset:   10,
	}, 5)

	sql, args := keeltest.Render(t, threadTitlesQuery(page))

	assert.Contains(t, sql, `SELECT at_."id", coalesce(at_."title", $5) FROM "assistant_threads" at_`, sql)
	assert.Contains(t, sql, `at_."id" IN (SELECT cs."thread_id" FROM "conversation_schedules" cs`, sql)
	assert.Contains(t, sql, `ORDER BY cs."created_at" DESC, cs."id" DESC LIMIT 5 OFFSET 10)`, sql)
	assert.Contains(t, args, threadID.String())
	assert.Equal(t, "", args[4])
}

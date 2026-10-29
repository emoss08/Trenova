package assistantturnrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestGetByIDQueryChecksTheOwnerWhenGiven(t *testing.T) {
	t.Parallel()

	owned, args := keeltest.Render(t, getByIDQuery(&assistantturnports.GetAssistantTurnRequest{
		ID:     pulid.ID("atrn_1"),
		UserID: pulid.ID("usr_1"),
	}))
	assert.Contains(t, owned, `"user_id" = `)
	assert.Contains(t, args, "usr_1")

	anyone, _ := keeltest.Render(
		t,
		getByIDQuery(&assistantturnports.GetAssistantTurnRequest{ID: pulid.ID("atrn_1")}),
	)
	assert.NotContains(t, anyone, `"user_id" = `)
}

func TestActiveQueryReadsALiveTurn(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, activeQuery(&assistantturnports.ActiveAssistantTurnRequest{
		ThreadID: pulid.ID("ath_1"),
		UserID:   pulid.ID("usr_1"),
	}))
	assert.Contains(t, sql, `at_."status" IN ('Pending', 'Running')`)
	assert.Contains(t, args, "ath_1")
}

func TestLiveTurnQueriesJoinTheThreadOfTheSamePerson(t *testing.T) {
	t.Parallel()

	req := assistantturnports.ListLiveAssistantTurnsRequest{
		UserID:         pulid.ID("usr_1"),
		ExcludeOrigins: []conversation.ThreadOrigin{conversation.ThreadOrigin("Schedule")},
	}
	turns, _ := keeltest.Render(t, liveTurnsQuery(&req))
	assert.Contains(t, turns, `JOIN "assistant_threads" live_thread`)
	assert.Contains(t, turns, `"origin" <> ALL(`)

	titles, _ := keeltest.Render(t, liveThreadTitlesQuery(&req))
	assert.Contains(t, titles, `"title"`)
	assert.Contains(t, titles, " IN (SELECT")
}

func TestLiveThreadTitlesReadAnUnnamedConversationAsEmpty(t *testing.T) {
	t.Parallel()

	titles, _ := keeltest.Render(
		t,
		liveThreadTitlesQuery(&assistantturnports.ListLiveAssistantTurnsRequest{
			UserID: pulid.ID("usr_1"),
		}),
	)
	assert.Contains(t, titles, `COALESCE(`)
	assert.Contains(t, titles, `"title"`)
}

package assistantturnrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
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
	assert.Contains(t, titles, `coalesce(`)
	assert.Contains(t, titles, `"title"`)
}

func TestPersonSpokeQueryReadsOnlyThePersonsLaterTurnsInTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, personSpokeQuery(&agentwaitports.PersonSpokeSinceRequest{
		ThreadID:   pulid.ID("athr_1"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")},
		Since:      1_800_000_000,
	}))
	for _, column := range []string{
		`"organization_id" = `, `"business_unit_id" = `, `"thread_id" = `,
		`"created_at" > `, `"origin" = `,
	} {
		assert.Contains(t, sql, column)
	}
	assert.Contains(t, args, "org_1")
	assert.Contains(t, args, "athr_1")
	assert.Contains(t, sql, `'Person'`)
}

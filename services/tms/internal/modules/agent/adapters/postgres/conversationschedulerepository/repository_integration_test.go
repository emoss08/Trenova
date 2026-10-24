//go:build integration

package conversationschedulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule/conversationscheduledb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
The Desk's Scheduled page lists every schedule a person keeps across their
conversations: the ones that will fire soonest first, the paused ones after
them, each with the title of the conversation its answers post into. A
conversation's own cards still read only that conversation's schedules, and
nobody reads another person's.
*/
func TestList_ReadsAPersonsSchedulesByNextRunWithTheirConversationTitles(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	owner := data.User.ID

	thread := func(title string) *conversation.Thread {
		created := &conversation.Thread{
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			UserID:            owner,
			AgentDefinitionID: pulid.MustNew("agd_"),
			Title:             title,
		}
		require.NoError(t, conversationdb.Threads.Create(ctx, db, created))

		return created
	}
	receivables := thread("AR follow-ups")
	untitled := thread("")

	next := func(at int64) *int64 { return &at }
	schedule := func(
		in *conversation.Thread,
		user pulid.ID,
		prompt string,
		enabled bool,
		nextRunAt *int64,
	) *conversationschedule.Schedule {
		created := &conversationschedule.Schedule{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			ThreadID:       in.ID,
			UserID:         user,
			Prompt:         prompt,
			Cadence:        "Every Monday · 8:00 AM",
			CronExpression: "0 8 * * 1",
			Timezone:       "America/New_York",
			Enabled:        enabled,
			NextRunAt:      nextRunAt,
		}
		require.NoError(t, conversationscheduledb.Schedules.Create(ctx, db, created))

		return created
	}

	later := schedule(receivables, owner, "Invoices past 60 days", true, next(2_000_000_000))
	soonest := schedule(untitled, owner, "Loads without a POD", true, next(1_900_000_000))
	unslotted := schedule(receivables, owner, "Unbilled shipments", true, nil)
	pausedSoon := schedule(untitled, owner, "Expiring permits", false, next(1_800_000_000))
	pausedLater := schedule(receivables, owner, "Carrier insurance", false, next(2_100_000_000))
	theirs := schedule(receivables, pulid.MustNew("usr_"), "Someone else's", true, next(1_000))

	listed, err := repo.List(ctx, conversationscheduleports.ListConversationSchedulesRequest{
		UserID:     owner,
		TenantInfo: tenant,
		Order:      conversationscheduleports.ConversationScheduleOrderNextRun,
		Limit:      50,
	})
	require.NoError(t, err)

	assert.Equal(t, 5, listed.Total, "another person's schedule is not counted")
	assert.Equal(t,
		[]pulid.ID{soonest.ID, later.ID, unslotted.ID, pausedSoon.ID, pausedLater.ID},
		listedIDs(listed.Items),
		"the ones that fire come first, soonest first, one with no slot last; paused ones follow",
	)
	titles := make(map[pulid.ID]string, len(listed.Items))
	for _, item := range listed.Items {
		titles[item.ID] = item.ThreadTitle
	}
	assert.Equal(t, "AR follow-ups", titles[later.ID])
	assert.Equal(t, "", titles[soonest.ID], "an untitled conversation reads as no title, not an error")
	assert.NotContains(t, titles, theirs.ID)

	page, err := repo.List(ctx, conversationscheduleports.ListConversationSchedulesRequest{
		UserID:     owner,
		TenantInfo: tenant,
		Order:      conversationscheduleports.ConversationScheduleOrderNextRun,
		Limit:      2,
		Offset:     2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, page.Total)
	assert.Equal(t, []pulid.ID{unslotted.ID, pausedSoon.ID}, listedIDs(page.Items),
		"a later page continues the same order")

	inThread, err := repo.List(ctx, conversationscheduleports.ListConversationSchedulesRequest{
		UserID:     owner,
		ThreadID:   receivables.ID,
		TenantInfo: tenant,
		Limit:      50,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]pulid.ID{later.ID, unslotted.ID, pausedLater.ID},
		listedIDs(inThread.Items),
		"a conversation's cards read only its own schedules",
	)
	for _, item := range inThread.Items {
		assert.Equal(t, "AR follow-ups", item.ThreadTitle)
	}

	other := seedtest.SeedAdditionalTenant(t, ctx, db, "ZZ")
	elsewhere, err := repo.List(
		other.Ctx(ctx),
		conversationscheduleports.ListConversationSchedulesRequest{
			UserID:     owner,
			TenantInfo: other.Info(),
			Limit:      50,
		},
	)
	require.NoError(t, err)
	assert.Zero(t, elsewhere.Total, "another tenant reads none of this tenant's schedules")
	assert.Empty(t, elsewhere.Items)
}

func listedIDs(items []*conversationscheduleports.ListedConversationSchedule) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

/*
A conversation opened with a schedule has no reply to be named from until the
schedule first runs, so saving the schedule names it. One that already has a
name keeps it.
*/
func TestCreate_NamesOnlyAnUnnamedConversation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	thread := func(title string) *conversation.Thread {
		created := &conversation.Thread{
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			UserID:            data.User.ID,
			AgentDefinitionID: pulid.MustNew("agd_"),
			Title:             title,
		}
		require.NoError(t, conversationdb.Threads.Create(ctx, db, created))

		return created
	}
	schedule := func(in *conversation.Thread) {
		_, _, err := repo.Create(ctx, conversationscheduleports.CreateConversationScheduleRequest{
			Schedule: &conversationschedule.Schedule{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				ThreadID:       in.ID,
				UserID:         data.User.ID,
				Prompt:         "Send me a list of delivered shipments",
				Cadence:        "Every Friday · 10:15 AM",
				CronExpression: "15 10 * * 5",
				Timezone:       "America/Chicago",
				Enabled:        true,
			},
			Message: conversation.Message{
				Role:    conversation.RoleUser,
				Kind:    conversation.MessageKindSchedule,
				Content: "/schedule every Friday at 10:15, send me a list of delivered shipments",
			},
			ThreadTitle: "Send me a list of delivered shipments",
		})
		require.NoError(t, err)
	}
	title := func(in *conversation.Thread) string {
		stored, err := conversationdb.Threads.Get(ctx, db, in.ID)
		require.NoError(t, err)

		return stored.Title
	}

	unnamed := thread("")
	named := thread("Friday reports")
	schedule(unnamed)
	schedule(named)

	assert.Equal(t, "Send me a list of delivered shipments", title(unnamed))
	assert.Equal(t, "Friday reports", title(named), "a name the conversation has is kept")
}

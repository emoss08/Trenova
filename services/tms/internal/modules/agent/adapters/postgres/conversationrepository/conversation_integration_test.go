//go:build integration

package conversationrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/conversationrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx    context.Context
	other  context.Context
	repo   conversationports.ConversationRepository
	tenant pagination.TenantInfo
	userID pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "CV")

	return &harness{
		ctx:   data.Ctx(ctx),
		other: other.Ctx(ctx),
		repo: conversationrepository.New(conversationrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant: data.Info(),
		userID: data.User.ID,
	}
}

func (h *harness) thread(t *testing.T, origin conversation.ThreadOrigin) *conversation.Thread {
	t.Helper()

	created, err := h.repo.CreateThread(h.ctx, &conversation.Thread{
		OrganizationID:    h.tenant.OrgID,
		BusinessUnitID:    h.tenant.BuID,
		UserID:            h.userID,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            conversation.ThreadStatusActive,
		Origin:            origin,
	})
	require.NoError(t, err)

	return created
}

func TestAppendTurnNumbersAfterTheLastMessageAndKeepsItsStamps(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t, conversation.ThreadOriginPanel)

	first, err := h.repo.AppendTurn(h.ctx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "Where is SHP-1?", CreatedAt: 100},
			{Role: conversation.RoleAssistant, Content: "In transit."},
		},
	})
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, 0, first[0].Sequence)
	assert.Equal(t, 1, first[1].Sequence)

	second, err := h.repo.AppendTurn(h.ctx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
		Messages:   []conversation.Message{{Role: conversation.RoleUser, Content: "Thanks"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, second[0].Sequence)

	messages, err := h.repo.ListMessages(h.ctx, &conversationports.ListMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	require.Len(t, messages, 3)
	assert.Equal(t, int64(100), messages[0].CreatedAt, "a stamp the caller gave is kept")

	latest, err := h.repo.ListMessages(h.ctx, &conversationports.ListMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
		Limit:      2,
	})
	require.NoError(t, err)
	require.Len(t, latest, 2)
	assert.Equal(t, 1, latest[0].Sequence, "the newest messages, oldest first")

	elsewhere, err := h.repo.CountMessages(
		h.other,
		conversationports.CountMessagesRequest{ThreadID: thread.ID},
	)
	require.NoError(t, err)
	assert.Zero(t, elsewhere, "another tenant counts none of the thread's messages")
}

func TestListThreadsPagesThroughEveryConversationOnce(t *testing.T) {
	h := newHarness(t)
	for range 5 {
		h.thread(t, conversation.ThreadOriginPanel)
	}
	h.thread(t, conversation.ThreadOriginAsk)

	seen := make(map[pulid.ID]struct{})
	cursor := ""
	for {
		page, err := h.repo.ListThreads(h.ctx, &conversationports.ListThreadsRequest{
			UserID:     h.userID,
			TenantInfo: h.tenant,
			Limit:      2,
			Cursor:     cursor,
		})
		require.NoError(t, err)
		for _, thread := range page.Items {
			_, repeated := seen[thread.ID]
			require.False(t, repeated, "a page never repeats a conversation")
			seen[thread.ID] = struct{}{}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	assert.Len(t, seen, 5, "unlisted conversations stay off the rail")

	other, err := h.repo.ListThreads(
		h.other,
		&conversationports.ListThreadsRequest{UserID: h.userID},
	)
	require.NoError(t, err)
	assert.Empty(t, other.Items)
}

func TestDeleteStaleThreadsRemovesTheThreadsAndTheirMessages(t *testing.T) {
	h := newHarness(t)
	stale := h.thread(t, conversation.ThreadOriginAsk)
	kept := h.thread(t, conversation.ThreadOriginPanel)
	_, err := h.repo.AppendTurn(h.ctx, &conversationports.AppendTurnRequest{
		ThreadID:   stale.ID,
		TenantInfo: h.tenant,
		Messages:   []conversation.Message{{Role: conversation.RoleUser, Content: "Old question"}},
	})
	require.NoError(t, err)

	deleted, err := h.repo.DeleteStaleThreads(h.ctx, conversationports.DeleteStaleThreadsRequest{
		Origin: conversation.ThreadOriginAsk,
		Before: stale.CreatedAt + 3_600,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	count, err := h.repo.CountMessages(
		h.ctx,
		conversationports.CountMessagesRequest{ThreadID: stale.ID},
	)
	require.NoError(t, err)
	assert.Zero(t, count)

	_, err = h.repo.GetThreadOwned(h.ctx, conversationports.GetThreadOwnedRequest{ID: kept.ID})
	require.NoError(t, err)
}

func TestUpdateThreadRefusesAStaleVersion(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t, conversation.ThreadOriginPanel)

	renamed := *thread
	renamed.Title = "Lane review"
	saved, err := h.repo.UpdateThread(h.ctx, &renamed)
	require.NoError(t, err)
	assert.Equal(t, thread.Version+1, saved.Version)

	stale := *thread
	stale.Title = "Stale"
	_, err = h.repo.UpdateThread(h.ctx, &stale)
	require.Error(t, err)

	read, err := h.repo.GetThread(
		h.ctx,
		&conversationports.GetThreadRequest{ID: thread.ID, UserID: h.userID},
	)
	require.NoError(t, err)
	assert.Equal(t, "Lane review", read.Title)
}
